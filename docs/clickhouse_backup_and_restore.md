# Clickhouse Cluster Backup And Restore

## Prerequisites

```
1. Assume we have `clickhouse-operator` already installed and running
```

### 1) Add Helm Repository(if not)

```bash
$ helm repo add ck https://radondb.github.io/radondb-clickhouse-operator/
$ helm repo update


$ helm repo list
NAME       	URL                                                     
stable     	https://charts.kubesphere.io/mirror                     
main       	https://charts.kubesphere.io/main                       
test       	https://charts.kubesphere.io/test                       
ck         	https://radondb.github.io/radondb-clickhouse-operator/
```

## 2) Install RadonDB ClickHouse Cluster

- 1、download `values.yaml` file

```bash
$ wget https://github.com/radondb/radondb-clickhouse-operator/blob/main/clickhouse-cluster/values.yaml
```

- 2、Modify the `backup` parameter of the `values.yaml` file

```yaml
# Configuration for the clickhouse backup
backup:
  # true for enable backup, false for disable
  enable: true

  # ClickHouse-backup image configuration
  image: radondb/clickhouse-backup:latest
  imagePullPolicy: IfNotPresent

  # Object storage related configuration required for backup
  # object storage must be compatible with AWS S3 protocol
  s3EndPoint: "http://s3.pek3.qingstor.com"
  s3Bucket: "ck"
  s3Path: "backup"
  s3AccessKey: "PFDHBVEIQIZYYGNTXSJY"
  s3SecretKey: "7f3SBspTALsdw2bXmuD125aLcxwdCvSdsMs03NkV"

  # The number of backups to keep, beyond which stale backups will be deleted
  localBackupKeep: 3
  remoteBackupKeep: 0
```

> note:
>
> 1、Please modify s3EndPoint, s3Bucket, s3Path, s3AccessKey, s3SecretKey to your s3 address.
>
> 2、s3Bucket must exist on your s3.
>
> 3、s3Path may not exist.

- 3、Install RadonDB ClickHouse Cluster

```
$ helm install --generate-name <repoName>/clickhouse-cluster -n <nameSpace>\
  --set <para_name>=<para_value>
```

Expected output

```bash
$ helm install clickhouse ck/clickhouse-cluster -n test
NAME: clickhouse
LAST DEPLOYED: Wed Aug 17 14:48:12 2021
NAMESPACE: test
STATUS: deployed
REVISION: 1
TEST SUITE: None
```



## 3) backup

> **Only MergeTree family tables engines**

### 1.signal

> signal will perform a backup.

```yaml
$ cat chb.yaml
apiVersion: clickhouse.radondb.com/v1
kind: ClickHouseBackup
metadata:
  name: chb
spec:
  chiName: clickhouse
  clusterName: all-nodes
  namespace: test
  backup:
    kind: single
```

apply this yaml:

` kubectl apply -f chb.yaml -n test `

get backup status：

```bash
$ kubectl get chbackup -n test -o wide
NAME   STATUS      STARTTIME             ENDTIME               BACKUPNAME                BACKUPSIZE   SCHEDULE
chb    Completed   2022-06-27-16:36:18   2022-06-27-16:36:38   chb-2022-06-27-16:36:18   0.00B
```

- 1、NAME：name
- 2、STATUS：Backup status，`Completed ` means the run is complete
- 3、STARTTIME：Backup start time
- 4、ENDTIME：Backup end time
- 5、BACKUPNAME：Backup name
- 6、BACKUPSIZE：Backup size
- 7、SCHEDULE：Backup schedule

### 2.schedule

> schedule will perform backups according to the specified schedule.

```yaml
apiVersion: clickhouse.radondb.com/v1
kind: ClickHouseBackup
metadata:
  name: chb-cron
spec:
  chiName: clickhouse
  clusterName: all-nodes
  namespace: test
  backup:
    kind: schedule
    schedule: "30 * * * *"
```

apply this yaml:

` kubectl apply -f chb.yaml -n test `

get backup status：

```bash
$ kubectl get chbackup -n test -o wide
NAME       STATUS      STARTTIME             ENDTIME               BACKUPNAME   BACKUPSIZE   SCHEDULE
chb-cron   Completed   2022-06-28-09:36:17   2022-06-28-09:36:17                             30 * * * *
```



The cron expression is as follows：

|    Field     | Required |     Values      | Characters |
| :----------: | :------: | :-------------: | :--------: |
|   Minutes    |   Yes    |      0-59       |  * / , -   |
|    Hours     |   Yes    |      0-23       |  * / , -   |
| Day of month |   Yes    |      1-31       |  * / , -   |
|    Month     |   Yes    | 1-12 or JAN-DEC |  * / , -   |
| Day of week  |    No    |       0-6       | * / , – ?  |

1) If the Day of week field is not provided, it is equivalent to *



eg：
1. Execute once a day at 23:00: `0 23 * * ?`
2. 1:00 every weekend: `0 1 * * 0`
3. Executed at 1:00 AM on the first day of each month : `0 1 1 * ?`
4. Execute once at 15 minutes, 30 minutes, and 45 minutes: `15,30,45 * * * ?`
5. Execute once every day at 0:00, 13:00: `0 0,13 * * ?`



## 4) restore

> Restore data to the cluster from backup.
>
> It is necessary to ensure that the backup cluster and the recovery cluster have the same number of replicas and shards.

- 1、get backup name

```
$ kubectl get chbackup -n test -o wide
NAME   STATUS      STARTTIME             ENDTIME               BACKUPNAME                BACKUPSIZE   SCHEDULE
chb    Completed   2022-06-27-16:36:18   2022-06-27-16:36:38   chb-2022-06-27-16:36:18   0.00B
```

- 2、Modify the `.spec.restore.backupName ` of the following yaml

```yaml
apiVersion: clickhouse.radondb.com/v1
kind: ClickHouseBackup
metadata:
  name: chb-restore
spec:
  chiName: clickhouse
  clusterName: all-nodes
  namespace: test
  restore:
    backupName: chb-2022-06-27-16:36:18
```

- 3、apply this yaml:

` kubectl apply -f chb.yaml -n test ` 