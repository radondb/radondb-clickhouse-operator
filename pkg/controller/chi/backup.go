package chi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/robfig/cron/v3"

	log "github.com/radondb/clickhouse-operator/pkg/announcer"
	v1 "github.com/radondb/clickhouse-operator/pkg/apis/clickhouse.radondb.com/v1"
	"github.com/radondb/clickhouse-operator/pkg/model"
	"github.com/radondb/clickhouse-operator/pkg/util"
)

type CronJob struct {
	worker *worker
	ctx    context.Context
	chb    *v1.ClickHouseBackup
}

func (c *CronJob) Run() {
	w := c.worker
	ctx := c.ctx
	chb := c.chb

	startTime := time.Now().Format(v1.TIME_LAYOUT)
	backupName := chb.Name + "-" + startTime

	(&chb.ChbStatus).ReconcileBackupRunning(startTime)
	_ = w.c.updateCHBObjectStatus(ctx, chb, false)

	if err := w.startSingleBackup(ctx, chb, backupName); err != nil {
		(&c.chb.ChbStatus).ReconcileBackupFailed(time.Now().Format(v1.TIME_LAYOUT))
		_ = c.worker.c.updateCHBObjectStatus(ctx, chb, false)
		return
	}
	(&chb.ChbStatus).ReconcileBackupCompleted(backupName, chb.ChbStatus.Schedule, time.Now().Format(v1.TIME_LAYOUT), w.getBackupSize(ctx, chb, backupName))
	_ = c.worker.c.updateCHBObjectStatus(ctx, chb, false)
}

func (w *worker) startCronBackup(ctx context.Context, chb *v1.ClickHouseBackup) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	c := cron.New()

	if _, err := c.AddJob(chb.Spec.Backup.Schedule, &CronJob{
		worker: w,
		ctx:    ctx,
		chb:    chb,
	}); err != nil {
		w.a.V(1).M(chb).F().Error("AddJob error, err = %v", err)
	}

	c.Start()

	w.c.AddCron(chb.Name, c)

	return nil
}

func (w *worker) stopCronBackup(ctx context.Context, chb *v1.ClickHouseBackup) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	if c := w.c.GetCron(chb.Name); c != nil {
		c.Stop()
		w.c.DeleteCron(chb.Name)
	}

	return nil
}

func (w *worker) startSingleBackup(ctx context.Context, chb *v1.ClickHouseBackup, backupName string) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}
	//var backupSize uint64

	if err := chb.WalkHost(func(host *v1.ChiHost, shardIndex, replicaIndex int) error {
		backupname := backupName + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

		if created, location := w.schemer.HostGetBackup(ctx, host, backupname); created != "" {
			w.a.V(1).M(chb).F().Info("backup %s already created at %s, located on %s", backupname, created, location)
			return errors.New("backup already exists")
		}

		return nil
	}); err != nil {
		return err
	}

	if err := chb.WalkHost(func(host *v1.ChiHost, shardIndex, replicaIndex int) error {
		hostname := model.CreatePodHostname(host)
		backupname := backupName + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

		var err error
		if 0 == replicaIndex {
			err = w.schemer.HostCreateBackup(ctx, host, backupname, "")
		} else {
			err = w.schemer.HostCreateBackup(ctx, host, backupname, "--engine=not,ReplicatedMergeTree")
		}

		if err != nil {
			w.a.V(1).M(chb).F().Error("error create %s on %s: %v", backupname, hostname, err)
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	if err := chb.WalkHost(func(host *v1.ChiHost, shardIndex, replicaIndex int) error {
		hostname := model.CreatePodHostname(host)
		backupname := backupName + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

		count := 0
		err := ""
		status := "in progress"
		for status == "in progress" {
			util.WaitContextDoneOrTimeout(ctx, time.Duration(5*(count+1))*time.Second)
			if 0 == replicaIndex {
				status, err = w.schemer.HostGetCreateBackupStatus(ctx, host, backupname, "")
			} else {
				status, err = w.schemer.HostGetCreateBackupStatus(ctx, host, backupname, "--engine=not,ReplicatedMergeTree")
			}
			w.a.V(1).M(chb).F().Error("in progress create %s on %s", backupname, hostname)
		}
		if status != "success" {
			w.a.V(1).M(chb).F().Error("error create %s on %s: %s", backupname, hostname, err)
		} else {
			w.a.V(1).M(chb).F().Info("success create %s on %s", backupname, hostname)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := chb.WalkHost(func(host *v1.ChiHost, shardIndex, replicaIndex int) error {
		hostname := model.CreatePodHostname(host)
		backupname := backupName + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

		if err := w.schemer.HostUploadBackup(ctx, host, backupname); err != nil {
			w.a.V(1).M(chb).F().Error("error upload %s on %s: %v", backupname, hostname, err)
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	if err := chb.WalkHost(func(host *v1.ChiHost, shardIndex, replicaIndex int) error {
		hostname := model.CreatePodHostname(host)
		backupname := backupName + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

		count := 0
		err := ""
		status := "in progress"
		for status == "in progress" {
			util.WaitContextDoneOrTimeout(ctx, time.Duration(5*(count+1))*time.Second)
			status, err = w.schemer.HostGetUploadBackupStatus(ctx, host, backupname)
			w.a.V(1).M(chb).F().Info("in progress upload %s on %s", backupname, hostname)
		}
		if status != "success" {
			w.a.V(1).M(chb).F().Error("error upload %s on %s: %s", backupname, hostname, err)
		} else {
			w.a.V(1).M(chb).F().Info("success upload %s on %s", backupname, hostname)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (w *worker) startRestore(ctx context.Context, chb *v1.ClickHouseBackup) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	if err := chb.WalkHost(func(host *v1.ChiHost, shardIndex, replicaIndex int) error {
		hostname := model.CreatePodHostname(host)
		backupname := chb.Spec.Restore.BackupName + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

		_, location := w.schemer.HostGetBackup(ctx, host, backupname)

		if location == "" {
			w.a.V(1).M(chb).F().Error("backup %s does not exists", backupname)
			return errors.New("backup does not exists")
		}
		if location == "local" {
			w.a.V(1).M(chb).F().Info("backup %s exists on local, skip download", backupname)
			return nil

		} else if location == "remote" {
			w.a.V(1).M(chb).F().Info("backup %s exists on remote, download it", backupname)

			if err := w.schemer.HostDownloadBackup(ctx, host, backupname); err != nil {
				w.a.V(1).M(chb).F().Error("error download %s on %s: %v", backupname, hostname, err)
				return err
			}

			count := 0
			err := ""
			status := "in progress"
			for status == "in progress" {
				util.WaitContextDoneOrTimeout(ctx, time.Duration(5*(count+1))*time.Second)
				status, err = w.schemer.HostGetDownloadBackupStatus(ctx, host, backupname)
			}
			if status != "success" {
				w.a.V(1).M(chb).F().Error("error download %s on %s: %s", backupname, hostname, err)
			} else {
				w.a.V(1).M(chb).F().Info("success download %s on %s", backupname, hostname)
			}
		}

		return nil
	}); err != nil {
		return err
	}

	if err := chb.WalkHost(func(host *v1.ChiHost, shardIndex, replicaIndex int) error {
		hostname := model.CreatePodHostname(host)
		backupname := chb.Spec.Restore.BackupName + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

		var err error
		if 0 == replicaIndex {
			err = w.schemer.HostRestoreBackup(ctx, host, backupname, "")
		} else {
			err = w.schemer.HostRestoreBackup(ctx, host, backupname, "--data")
		}

		if err != nil {
			w.a.V(1).M(chb).F().Error("error restore %s on %s: %v", backupname, hostname, err)
			return err
		}

		count := 0
		errs := ""
		status := "in progress"
		for status == "in progress" {
			util.WaitContextDoneOrTimeout(ctx, time.Duration(5*(count+1))*time.Second)
			if 0 == replicaIndex {
				status, errs = w.schemer.HostGetRestoreBackupStatus(ctx, host, backupname, "")
			} else {
				status, errs = w.schemer.HostGetRestoreBackupStatus(ctx, host, backupname, "--data")
			}
		}
		if status != "success" {
			w.a.V(1).M(chb).F().Error("error restore %s on %s: %s", backupname, hostname, errs)
		} else {
			w.a.V(1).M(chb).F().Info("success restore %s on %s", backupname, hostname)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// addStrings calculate the sum of strings
func addStrings(num1 string, num2 string) string {
	add := 0
	ans := ""
	for i, j := len(num1) - 1, len(num2) - 1; i >= 0 || j >= 0 || add != 0; i, j = i - 1, j - 1 {
		var x, y int
		if i >= 0 {
			x = int(num1[i] - '0')
		}
		if j >= 0 {
			y = int(num2[j] - '0')
		}
		result := x + y + add
		ans = strconv.Itoa(result%10) + ans
		add = result / 10
	}
	return ans
}

// formatFileSize unit conversion, two decimal
func formatFileSize(fileSize uint64) string {
	var unit string
	var size int

	if fileSize < v1.KB {
		unit, size = "B", v1.B
	} else if fileSize < (v1.MB) {
		unit, size = "KB", v1.KB
	} else if fileSize < (v1.GB) {
		unit, size = "MB", v1.MB
	} else if fileSize < (v1.TB) {
		unit, size = "GB", v1.GB
	} else if fileSize < (v1.PB) {
		unit, size = "TB", v1.TB
	} else {
		unit, size = "PB", v1.PB
	}
	return fmt.Sprintf("%.2f%s", float64(fileSize)/float64(size), unit)
}

// getBackupSize
func (w *worker) getBackupSize(ctx context.Context, chb *v1.ClickHouseBackup, backupName string) string {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return ""
	}

	backupSize := "0"
	// get backupup size
	if err := chb.WalkHost(func(host *v1.ChiHost, shardIndex, replicaIndex int) error {
		backupname := backupName + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

		created, size := w.schemer.HostGetBackupSize(ctx, host, backupname)

		if created == "" {
			w.a.V(1).M(chb).F().Error("backup %s does not exists", backupname)
			return errors.New("backup does not exists")
		}
		backupSize = addStrings(backupSize, size)

		return nil
	}); err != nil {
		return ""
	}

	size, _ := strconv.ParseUint(backupSize, 10, 64)
	return formatFileSize(size)
}

func (w *worker) startBackupOrRestore(ctx context.Context, chb *v1.ClickHouseBackup, backupName string) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	if !chb.Spec.Restore.IsEmpty() {
		if err := w.startRestore(ctx, chb); err != nil {
			w.a.V(1).M(chb).A().Error("unable to start restore: %s/%v", chb.Name, err)
			return err
		}
	} else if !chb.Spec.Backup.IsEmpty() {
		switch chb.Spec.Backup.Kind {
		case v1.ClickHouseBackupKindSingle:
			if err := w.startSingleBackup(ctx, chb, backupName); err != nil {
				w.a.V(1).M(chb).A().Error("unable to start backup: %s/%v", chb.Name, err)
				return err
			}
		case v1.ClickHouseBackupKindSchedule:
			if err := w.startCronBackup(ctx, chb); err != nil {
				w.a.V(1).M(chb).A().Error("unable to start schedule backup: %s/%v", chb.Name, err)
				return err
			}
		default:
			w.a.V(1).F().Error("ClickHouseBackup backup kind %s do not support", chb.Spec.Backup.Kind)
			(&chb.ChbStatus).ReconcileBackupUnknow(time.Now().Format(v1.TIME_LAYOUT))
		}
	} else {
		w.a.V(1).F().Warning("Backup & restore is not specified, nothing to do.")
		(&chb.ChbStatus).ReconcileBackupUnknow(time.Now().Format(v1.TIME_LAYOUT))
	}

	return nil
}