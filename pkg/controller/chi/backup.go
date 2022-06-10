package chi

import (
	"context"
	"errors"
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
	_ = c.worker.startSingleBackup(c.ctx, c.chb)
}

func (w *worker) startCronBackup(ctx context.Context, chb *v1.ClickHouseBackup) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	c := cron.New()

	if schedule, err := cron.ParseStandard(chb.Spec.Backup.Schedule); err != nil {
		c.Schedule(schedule, &CronJob{
			worker: w,
			ctx:    ctx,
			chb:    chb,
		})
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

func (w *worker) startSingleBackup(ctx context.Context, chb *v1.ClickHouseBackup) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	if err := chb.WalkHost(func(host *v1.ChiHost, shardIndex, replicaIndex int) error {
		backupname := chb.Name + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

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
		backupname := chb.Name + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

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
		backupname := chb.Name + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

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
		backupname := chb.Name + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

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
		backupname := chb.Name + "-" + strconv.Itoa(shardIndex) + "-" + strconv.Itoa(replicaIndex)

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
