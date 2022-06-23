# Clickhouse Cluster Backup And Restore

## Prerequisites

```
1. Assume we have `clickhouse-operator` already installed and running
```





### 1) 添加 helm 源

```bash
helm repo add ck-operator https://radondb.github.io/radondb-clickhouse-operator/
helm repo update


root@i-7oefo0ca:~# helm repo list
NAME       	URL                                                     
stable     	https://charts.kubesphere.io/mirror                     
main       	https://charts.kubesphere.io/main                       
test       	https://charts.kubesphere.io/test                       
ck-operator	https://radondb.github.io/radondb-clickhouse-operator/
```

## 2) deploy clickhouse cluster

1. 下载 `values.yaml` 文件。

```
wget https://github.com/radondb/radondb-clickhouse-kubernetes/blob/main/clickhouse-cluster/values.yaml
```

2. 修改 `values.yaml` 文件为以下内容

```
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

> note: 请修改 s3EndPoint、s3Bucket、s3Path、s3AccessKey、s3SecretKey 为您的 s3 地址。
>
> s3Bucket 必须在您的 s3 上存在。
>
> s3Path 可以不存在。

3. 执行如下命令，部署集群。

```
$ helm install --generate-name <repoName>/clickhouse-cluster -n <nameSpace>\
  --set <para_name>=<para_value>
```

预期效果：

```bash
$ helm install clickhouse ck-operator/clickhouse-cluster -n test
NAME: clickhouse
LAST DEPLOYED: Mon Jun  6 10:50:29 2022
NAMESPACE: test
STATUS: deployed
REVISION: 1
TEST SUITE: None
```



## 3) backup

### 1.signal

> signal 会执行一次备份。

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

执行如下命令，应用此 yaml:

` kubectl apply -f chb.yaml -n test `







### 2.schedule

> schedule 会按照指定的 schedule 定时执行备份。

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



The cron expression is as follows：

|    字段名    | 是否必须 |    允许的值     | 允许的字符 |
| :----------: | :------: | :-------------: | :--------: |
|   Minutes    |    是    |      0-59       |  * / , -   |
|    Hours     |    是    |      0-23       |  * / , -   |
| Day of month |    是    |      1-31       |  * / , -   |
|    Month     |    是    | 1-12 or JAN-DEC |  * / , -   |
| Day if week  |    否    |       0-6       | * / , – ?  |

1) If the Day of week field is not provided, it is equivalent to *



eg：
1. Execute once a day at 23:00: `0 23 * * ?`
2. 1:00 every weekend: `0 1 * * 6`(未测试)
3. Executed at 1:00 AM on the first day of each month : `0 1 1 * ?`
4. Execute once at 15 minutes, 30 minutes, and 45 minutes: `15,30,45 * * * ?`
5. Execute once every day at 0:00, 13:00: `0 0,13 * * ?`



## 4) restore

> 从备份中恢复数据到集群。
>
> 需要保证备份集群和恢复集群副本及分片数相同。

- 1、get backup name

```
kubectl get chbackup -n <namespace>
```

- 2、apply yaml

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
    backupName: chb-2022-06-09-08-26-57
```