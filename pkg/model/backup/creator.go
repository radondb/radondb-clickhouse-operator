package backup

import (
	"bytes"
	"strconv"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/radondb/clickhouse-operator/pkg/announcer"
	chiv1 "github.com/radondb/clickhouse-operator/pkg/apis/clickhouse.radondb.com/v1"
	"github.com/radondb/clickhouse-operator/pkg/chop"
	"github.com/radondb/clickhouse-operator/pkg/model"
	"github.com/radondb/clickhouse-operator/pkg/util"
)

// Creator specifies creator object
type Creator struct {
	chb *chiv1.ClickHouseBackup
	a   log.Announcer
}

// NewCreator creates new Creator object
func NewCreator(chb *chiv1.ClickHouseBackup) *Creator {
	return &Creator{
		chb: chb,
		a:   log.M(chb),
	}
}

// CreateJobCHBRestore creates new batchv1.CronJob for restore clickhouse
func (c *Creator) CreateJobCHBRestore() *batchv1.Job {
	if c.chb.Spec.Restore.IsEmpty() {
		c.a.V(1).F().Error("no restore specify")
		return nil
	}

	c.a.V(1).F().Info("ClickHouseBackup Restore Job %s/%s", c.chb.Namespace, c.chb.Name)

	// Create default Job
	if podSpec := c.getPodSpec(chiv1.ClickHouseBackupRestore); podSpec == nil {
		log.V(1).F().Error("unknown error when get pod spec")
		return nil
	} else {
		return &batchv1.Job{
			ObjectMeta: *c.getObjectMeta(),
			Spec:       *podSpec,
		}
	}
}

// CreateJobCHBBackup creates new batchv1.CronJob for backup clickhouse
func (c *Creator) CreateJobCHBBackup() *batchv1.Job {
	if !c.chb.Spec.Backup.IsSingleBackup() {
		c.a.Error("not single backup kind")
		return nil
	}

	if c.chb.Cluster == nil {
		c.a.Error("no cluster spec")
		return nil
	}

	c.a.V(1).F().Info("ClickHouseBackup Backup Job %s/%s", c.chb.Namespace, c.chb.Name)

	// Create default Job
	if podSpec := c.getPodSpec(chiv1.ClickHouseBackupBackup); podSpec == nil {
		log.V(1).F().Error("unknown error when get pod spec")
		return nil
	} else {
		return &batchv1.Job{
			ObjectMeta: *c.getObjectMeta(),
			Spec:       *podSpec,
		}
	}

}

// CreateCronJobCHBBackup creates new batchv1.CronJob for backup clickhouse
func (c *Creator) CreateCronJobCHBBackup() *batchv1beta1.CronJob {
	if !c.chb.Spec.Backup.IsScheduleBackup() {
		c.a.Error("not schedule backup kind")
		return nil
	}

	if c.chb.Cluster == nil {
		c.a.Error("no cluster spec")
		return nil
	}

	c.a.V(1).F().Info("ClickHouseBackup Backup CronJob %s/%s", c.chb.Namespace, c.chb.Name)

	// Create default Job
	if podSpec := c.getPodSpec(chiv1.ClickHouseBackupBackup); podSpec == nil {
		log.V(1).F().Error("unknown error when get pod spec")
		return nil
	} else {
		return &batchv1beta1.CronJob{
			ObjectMeta: *c.getObjectMeta(),
			Spec: batchv1beta1.CronJobSpec{
				Schedule:          c.chb.Spec.Backup.Schedule,
				ConcurrencyPolicy: batchv1beta1.ForbidConcurrent,
				JobTemplate: batchv1beta1.JobTemplateSpec{
					Spec: *podSpec,
				},
			},
		}
	}
}

// getObjectMeta get metav1.ObjectMeta for Job / CronJob
func (c *Creator) getObjectMeta() *metav1.ObjectMeta {
	return &metav1.ObjectMeta{
		Name:      c.chb.Name,
		Namespace: c.chb.Spec.Namespace,
		Labels:    nil,
	}
}

// getPodSpec get batchv1.JobSpec for Job / CronJob
func (c *Creator) getPodSpec(kind string) *batchv1.JobSpec {
	backoffLimit := int32(1)
	parallelism := int32(1)
	completions := int32(1)

	var command string
	switch kind {
	case chiv1.ClickHouseBackupBackup:
		command = c.getBackupCommand()
	case chiv1.ClickHouseBackupRestore:
		command = c.getRestoreCommand()
	default:
		log.V(1).F().Error("ClickHouseBackup kind %s do not support", kind)
		return nil
	}

	return &batchv1.JobSpec{
		BackoffLimit: &backoffLimit,
		Parallelism:  &parallelism,
		Completions:  &completions,
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyNever,
				Containers: []corev1.Container{
					{
						Name:            "clickhouse-client",
						Image:           c.chb.Spec.Image,
						ImagePullPolicy: corev1.PullPolicy(c.chb.Spec.ImagePullPolicy),
						Env: []corev1.EnvVar{
							{
								Name:  "CLICKHOUSE_PORT",
								Value: strconv.Itoa(int(c.chb.Spec.TCPPort)),
							},
						},
						Command: []string{
							"bash",
							"-ec",
							command,
						},
					},
				},
			},
		},
	}
}

// getBackupCommand get backup command to back up cluster.
// Due to the characteristics of replicatedMergeTree,
// in the same shard, the data of replicatedMergeTree can only be restored once.
// So in the same shard, we only choose one replica to back up all data,
// and the other replica to back up only the data except replicatedMergeTree.
func (c *Creator) getBackupCommand() string {
	b := &bytes.Buffer{}
	util.Iline(b, 0, "BACKUP_DATE=$(date +%%Y-%%m-%%d-%%H-%%M-%%S);")
	util.Iline(b, 0, "BACKUP_NAME_MAIN="+c.chb.Name+"-$BACKUP_DATE;")
	util.Iline(b, 0, "BACKUP_USER="+chop.Config().CHUsername+";")
	util.Iline(b, 0, "BACKUP_PASSWORD="+chop.Config().CHPassword+";")

	for shardIndex := range c.chb.Cluster.Layout.Shards {
		shard := &c.chb.Cluster.Layout.Shards[shardIndex]
		for replicaIndex := range shard.Hosts {
			host := shard.Hosts[replicaIndex]
			hostname := model.CreatePodHostname(host)

			util.Iline(b, 0, "CLICKHOUSE_SERVER="+hostname+";")
			util.Iline(b, 0, "BACKUP_NAME=$BACKUP_NAME_MAIN-"+strconv.Itoa(shardIndex)+"-"+strconv.Itoa(replicaIndex)+";")

			if 0 == replicaIndex {
				util.Iline(b, 0, "clickhouse-client --echo -mn -q \"INSERT INTO system.backup_actions(command) VALUES('create ${BACKUP_NAME}')\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")
			} else {
				util.Iline(b, 0, "clickhouse-client --echo -mn -q \"INSERT INTO system.backup_actions(command) VALUES('create ${BACKUP_NAME} --engine=not,ReplicatedMergeTree')\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")
			}
		}
	}

	for shardIndex := range c.chb.Cluster.Layout.Shards {
		shard := &c.chb.Cluster.Layout.Shards[shardIndex]
		for replicaIndex := range shard.Hosts {
			host := shard.Hosts[replicaIndex]
			hostname := model.CreatePodHostname(host)

			util.Iline(b, 0, "CLICKHOUSE_SERVER="+hostname+";")
			util.Iline(b, 0, "BACKUP_NAME=$BACKUP_NAME_MAIN-"+strconv.Itoa(shardIndex)+"-"+strconv.Itoa(replicaIndex)+";")

			util.Iline(b, 0, "while [[ 'in progress' == $(clickhouse-client -mn -q \"SELECT status FROM system.backup_actions WHERE command like 'create%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD) ]]; do")
			util.Iline(b, 0, "  echo \"still in progress $BACKUP_NAME on $CLICKHOUSE_SERVER\";")
			util.Iline(b, 0, "  sleep 1;")
			util.Iline(b, 0, "done;")
			util.Iline(b, 0, "if [[ 'success' != $(clickhouse-client -mn -q \"SELECT status FROM system.backup_actions WHERE command like 'create%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD) ]]; then")
			util.Iline(b, 0, "  echo \"error create $BACKUP_NAME on $CLICKHOUSE_SERVER\";")
			util.Iline(b, 0, "  clickhouse-client -mn --echo -q \"SELECT status, error FROM system.backup_actions WHERE command like 'create%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")
			util.Iline(b, 0, "  exit 1;")
			util.Iline(b, 0, "fi;")
		}
	}

	for shardIndex := range c.chb.Cluster.Layout.Shards {
		shard := &c.chb.Cluster.Layout.Shards[shardIndex]
		for replicaIndex := range shard.Hosts {
			host := shard.Hosts[replicaIndex]
			hostname := model.CreatePodHostname(host)

			util.Iline(b, 0, "CLICKHOUSE_SERVER="+hostname+";")
			util.Iline(b, 0, "BACKUP_NAME=$BACKUP_NAME_MAIN-"+strconv.Itoa(shardIndex)+"-"+strconv.Itoa(replicaIndex)+";")

			util.Iline(b, 0, "echo \"upload $BACKUP_NAME on $CLICKHOUSE_SERVER\";")
			util.Iline(b, 0, "clickhouse-client --echo -mn -q \"INSERT INTO system.backup_actions(command) VALUES('upload ${BACKUP_NAME}')\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")
		}
	}

	for shardIndex := range c.chb.Cluster.Layout.Shards {
		shard := &c.chb.Cluster.Layout.Shards[shardIndex]
		for replicaIndex := range shard.Hosts {
			host := shard.Hosts[replicaIndex]
			hostname := model.CreatePodHostname(host)

			util.Iline(b, 0, "CLICKHOUSE_SERVER="+hostname+";")
			util.Iline(b, 0, "BACKUP_NAME=$BACKUP_NAME_MAIN-"+strconv.Itoa(shardIndex)+"-"+strconv.Itoa(replicaIndex)+";")

			util.Iline(b, 0, "while [[ 'in progress' == $(clickhouse-client -mn -q \"SELECT status FROM system.backup_actions WHERE command like 'upload%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD) ]]; do")
			util.Iline(b, 0, "  echo \"still in progress $BACKUP_NAME on $CLICKHOUSE_SERVER\";")
			util.Iline(b, 0, "  sleep 1;")
			util.Iline(b, 0, "done;")
			util.Iline(b, 0, "if [[ 'success' != $(clickhouse-client -mn -q \"SELECT status FROM system.backup_actions WHERE command like 'upload%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD) ]]; then")
			util.Iline(b, 0, "  echo \"error create $BACKUP_NAME on $CLICKHOUSE_SERVER\";")
			util.Iline(b, 0, "  clickhouse-client -mn --echo -q \"SELECT status, error FROM system.backup_actions WHERE command like 'upload%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")
			util.Iline(b, 0, "  exit 1;")
			util.Iline(b, 0, "fi;")
		}
	}

	util.Iline(b, 0, "echo 'BACKUP CREATED'")

	return b.String()
}

// getRestoreCommand create restore command to restore from backup.
// Since clickhouse on qingcloud will add default cluster to the DDL statement,
// when restoring data to the cluster, only one node needs to be selected for DDL statement recovery.
func (c *Creator) getRestoreCommand() string {
	b := &bytes.Buffer{}
	util.Iline(b, 0, "BACKUP_NAME_MAIN="+c.chb.Spec.Restore.BackupName+";")
	util.Iline(b, 0, "BACKUP_USER="+chop.Config().CHUsername+";")
	util.Iline(b, 0, "BACKUP_PASSWORD="+chop.Config().CHPassword+";")

	for shardIndex := range c.chb.Cluster.Layout.Shards {
		shard := &c.chb.Cluster.Layout.Shards[shardIndex]
		for replicaIndex := range shard.Hosts {
			host := shard.Hosts[replicaIndex]
			hostname := model.CreatePodHostname(host)

			util.Iline(b, 0, "CLICKHOUSE_SERVER="+hostname+";")
			util.Iline(b, 0, "BACKUP_NAME=$BACKUP_NAME_MAIN-"+strconv.Itoa(shardIndex)+"-"+strconv.Itoa(replicaIndex)+";")

			util.Iline(b, 0, "clickhouse-client --echo -mn -q \"INSERT INTO system.backup_actions(command) VALUES('download ${BACKUP_NAME}')\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")

			util.Iline(b, 0, "while [[ 'in progress' == $(clickhouse-client -mn -q \"SELECT status FROM system.backup_actions WHERE command like 'download%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD) ]]; do")
			util.Iline(b, 0, "  echo \"still in progress $BACKUP_NAME on $CLICKHOUSE_SERVER\";")
			util.Iline(b, 0, "  sleep 1;")
			util.Iline(b, 0, "done;")
			util.Iline(b, 0, "if [[ 'success' != $(clickhouse-client -mn -q \"SELECT status FROM system.backup_actions WHERE command like 'download%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD) ]]; then")
			util.Iline(b, 0, "  echo \"error create $BACKUP_NAME on $CLICKHOUSE_SERVER\";")
			util.Iline(b, 0, "  clickhouse-client -mn --echo -q \"SELECT status, error FROM system.backup_actions WHERE command like 'download%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")
			util.Iline(b, 0, "  exit 1;")
			util.Iline(b, 0, "fi;")
		}
	}

	for shardIndex := range c.chb.Cluster.Layout.Shards {
		shard := &c.chb.Cluster.Layout.Shards[shardIndex]
		for replicaIndex := range shard.Hosts {
			host := shard.Hosts[replicaIndex]
			hostname := model.CreatePodHostname(host)

			util.Iline(b, 0, "CLICKHOUSE_SERVER="+hostname+";")
			util.Iline(b, 0, "BACKUP_NAME=$BACKUP_NAME_MAIN-"+strconv.Itoa(shardIndex)+"-"+strconv.Itoa(replicaIndex)+";")

			if 0 == shardIndex && 0 == replicaIndex {
				util.Iline(b, 0, "clickhouse-client --echo -mn -q \"INSERT INTO system.backup_actions(command) VALUES('restore ${BACKUP_NAME}')\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")
			} else {
				util.Iline(b, 0, "clickhouse-client --echo -mn -q \"INSERT INTO system.backup_actions(command) VALUES('restore ${BACKUP_NAME} --data')\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")
			}

			util.Iline(b, 0, "while [[ 'in progress' == $(clickhouse-client -mn -q \"SELECT status FROM system.backup_actions WHERE command like 'restore%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD) ]]; do")
			util.Iline(b, 0, "  echo \"still in progress $BACKUP_NAME on $CLICKHOUSE_SERVER\";")
			util.Iline(b, 0, "  sleep 1;")
			util.Iline(b, 0, "done;")
			util.Iline(b, 0, "if [[ 'success' != $(clickhouse-client -mn -q \"SELECT status FROM system.backup_actions WHERE command like 'restore%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD) ]]; then")
			util.Iline(b, 0, "  echo \"error create $BACKUP_NAME on $CLICKHOUSE_SERVER\";")
			util.Iline(b, 0, "  clickhouse-client -mn --echo -q \"SELECT status, error FROM system.backup_actions WHERE command like 'restore%%${BACKUP_NAME}%%'\" --host=$CLICKHOUSE_SERVER --port=$CLICKHOUSE_PORT --user=$BACKUP_USER --password=$BACKUP_PASSWORD;")
			util.Iline(b, 0, "  exit 1;")
			util.Iline(b, 0, "fi;")

		}
	}

	util.Iline(b, 0, "echo 'BACKUP CREATED'")

	return b.String()
}
