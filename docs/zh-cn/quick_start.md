> [English](../quick_start.md) | 简体中文 

---

- [快速入门](#快速入门)
  - [前提条件](#前提条件)
  - [部署 RadonDB ClickHouse Operator](#部署-radondb-clickhouse-operator)
    - [场景 1：在 `kube-system` 命名空间部署](#场景-1在-kube-system-命名空间部署)
    - [场景 2：在 Kubernetes `1.17`及以下版本的 `kube-system` 命名空间中部署](#场景-2在-kubernetes-117及以下版本的-kube-system-命名空间中部署)
    - [场景 3：自定命名空间部署](#场景-3自定命名空间部署)
    - [场景 4：离线部署](#场景-4离线部署)
    - [验证 Operator 部署](#验证-operator-部署)
    - [从源码构建 Operator](#从源码构建-operator)
  - [部署 RadonDB ClickHouse 集群](#部署-radondb-clickhouse-集群)
    - [创建自定义命名空间](#创建自定义命名空间)
    - [示例 1：测试集群](#示例-1测试集群)
    - [示例 2：默认持久卷](#示例-2默认持久卷)
    - [示例 3：自定义部署 Pod 和 VolumeClaim](#示例-3自定义部署-pod-和-volumeclaim)
    - [示例 4：自定义 ClickHouse 配置](#示例-4自定义-clickhouse-配置)
    - [验证集群部署](#验证集群部署)
  - [访问 RadonDB ClickHouse](#访问-radondb-clickhouse)
    - [通过 EXTERNAL-IP 访问](#通过-external-ip-访问)
    - [通过 pod-NAME 访问](#通过-pod-name-访问)

# 快速入门

## 前提条件

- 已准备 Kubernetes 集群，且集群正常运行。
- 已准备 `kubectl` 工具，且正确配置。
- 已准备 `curl` 工具。

## 部署 RadonDB ClickHouse Operator

部署 `radondb-clickhouse-operator`，最简单的办法是直接从 `github` 应用部署示例。

### 场景 1：在 `kube-system` 命名空间部署

进入`kube-system` 命名空间，执行如下命令：

```bash
kubectl apply -f https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/deploy/operator/clickhouse-operator-install-bundle.yaml
```

### 场景 2：在 Kubernetes `1.17`及以下版本的 `kube-system` 命名空间中部署

进入`kube-system` 命名空间，执行如下命令：

```bash
kubectl apply -f https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/deploy/operator/clickhouse-operator-install-bundle-v1beta1.yaml
```

### 场景 3：自定命名空间部署

使用如下安装脚本，自定义 Operator 和 Operator 镜像的部署位置。

```bash
curl -s https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/deploy/operator-web-installer/clickhouse-operator-install.sh | OPERATOR_NAMESPACE=test-clickhouse-operator bash
```

如上命令，将新建一个命名空间，且该命名空间将被用于安装 `radondb-clickhouse-operator`。部署过程中将下载 `.yaml` 和 `.xml` 文件。安装成功后，将回显在 `test-clickhouse-operator` 命名空间成功部署自定义资源 `kind: ClickhouseInstallation`。

若未指定 `OPERATOR_NAMESPACE` 参数值，将部署至默认命名空间 `kube-system`，安装成功后，将回显在所有命名空间成功部署自定义资源 `kind: ClickhouseInstallation`。

```bash
cd ~
curl -s https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/deploy/operator-web-installer/clickhouse-operator-install.sh | bash
```

### 场景 4：离线部署

下载 [radondb-clickhouse-operator 部署模版文件](../deploy/operator/clickhouse-operator-install-template.yaml)，并自定义参数设置。通过 `kubectl` 工具应用该模版文件。

```bash
kubectl apply -f <your_file_path>/clickhouse-operator-install-template.yaml
```

或者通过 `kubectl` 工具直接执行如下脚本。

```bash
# Namespace to install operator into
OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE:-test-clickhouse-operator}"
# Namespace to install metrics-exporter into
METRICS_EXPORTER_NAMESPACE="${OPERATOR_NAMESPACE}"

# Operator's docker image
OPERATOR_IMAGE="${OPERATOR_IMAGE:-radondb/chronus-operator:latest}"
# Metrics exporter's docker image
METRICS_EXPORTER_IMAGE="${METRICS_EXPORTER_IMAGE:-radondb/chronus-metrics-operator:latest}"

# Setup clickhouse-operator into specified namespace
kubectl apply --namespace="${OPERATOR_NAMESPACE}" -f <( \
    curl -s https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/deploy/operator/clickhouse-operator-install-template.yaml | \
        OPERATOR_IMAGE="${OPERATOR_IMAGE}" \
        OPERATOR_NAMESPACE="${OPERATOR_NAMESPACE}" \
        METRICS_EXPORTER_IMAGE="${METRICS_EXPORTER_IMAGE}" \
        METRICS_EXPORTER_NAMESPACE="${METRICS_EXPORTER_NAMESPACE}" \
        envsubst \
)
```

### 验证 Operator 部署

参考以上样例，部署 RadonDB ClickHouse Operator 成功后，将回显如下示例信息。

```text
Setup ClickHouse Operator into test-clickhouse-operator namespace
namespace/test-clickhouse-operator created
customresourcedefinition.apiextensions.k8s.io/clickhouseinstallations.clickhouse.radondb.com configured
serviceaccount/clickhouse-operator created
clusterrolebinding.rbac.authorization.k8s.io/clickhouse-operator configured
service/clickhouse-operator-metrics created
configmap/etc-clickhouse-operator-files created
configmap/etc-clickhouse-operator-confd-files created
configmap/etc-clickhouse-operator-configd-files created
configmap/etc-clickhouse-operator-templatesd-files created
configmap/etc-clickhouse-operator-usersd-files created
deployment.apps/clickhouse-operator created
```

可执行如下命令，确认 `radondb-clickhouse-operator`是否正常运行。

```bash
$ kubectl get pods -n test-clickhouse-operator
NAME                                 READY   STATUS    RESTARTS   AGE
clickhouse-operator-5ddc6d858f-drppt 1/1     Running   0          1m
```

### 从源码构建 Operator

关于如何从源码构建 RadonDB ClickHouse Operator，以及如何部署一个 docker 镜像并在 `kubernetes` 中使用，请参考[从源码构建 Operator](../operator_build_from_sources.md)。

## 部署 RadonDB ClickHouse 集群

`radondb-clickhouse-operator` 项目提供多个[部署集群示例](../chi-examples/)，以下为部分示例说明。

### 创建自定义命名空间

为方便 RadonDB ClickHouse 集群管理和高效运行，推荐将所有集群组件放在同一命名空间。以下以创建一个 `test` 命名空间为例。

```bash
$ kubectl create namespace test-clickhouse-operator
namespace/test-clickhouse-operator created
```

### 示例 1：测试集群

以下创建一个 [1 shard 1 replica](../chi-examples/01-simple-layout-01-1shard-1repl.yaml) 测试集群为例。

> **注意**
> 
> 该集群不具备持久存储能力，仅用于部署验证。
 
```bash
$ kubectl apply -n test-clickhouse-operator -f https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/docs/chi-examples/01-simple-layout-01-1shard-1repl.yaml
clickhouseinstallation.clickhouse.radondb.com/simple-01 created
```

[1 shard 1 replica](../chi-examples/01-simple-layout-01-1shard-1repl.yaml) 定义了一个单副本 RadonDB ClickHouse 集群。

```yaml
apiVersion: "clickhouse.radondb.com/v1"
kind: "ClickHouseInstallation"
metadata:
  name: "simple-01"
```

### 示例 2：默认持久卷

[默认持久卷示例](../chi-examples/03-persistent-volume-01-default-volume.yaml) 定义了一个动态持久存储卷的 RadonDB ClickHouse 集群。

```bash
$ kubectl apply -n test-clickhouse-operator -f https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/docs/chi-examples/03-persistent-volume-01-default-volume.yaml
clickhouseinstallation.clickhouse.radondb.com/pv-simple created
```

```yaml
apiVersion: "clickhouse.radondb.com/v1"
kind: "ClickHouseInstallation"
metadata:
  name: "pv-simple"
spec:
  defaults:
    templates:
      dataVolumeClaimTemplate: volume-template
      logVolumeClaimTemplate: volume-template
  configuration:
    clusters:
      - name: "simple"
        layout:
          shardsCount: 1
          replicasCount: 1
      - name: "replicas"
        layout:
          shardsCount: 1
          replicasCount: 2
      - name: "shards"
        layout:
          shardsCount: 2
  templates:
    volumeClaimTemplates:
      - name: volume-template
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 123Mi
```

### 示例 3：自定义部署 Pod 和 VolumeClaim

[自定义部署 Pod 和 VolumeClaim 示例](../chi-examples/03-persistent-volume-02-volume.yaml) 定义了如下配置：

- 指定部署
- Pod 模版
- VolumeClaim 模版

```bash
$ kubectl apply -n test-clickhouse-operator -f https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/docs/chi-examples/03-persistent-volume-02-volume.yaml
clickhouseinstallation.clickhouse.radondb.com/pv-log created
```

```yaml
apiVersion: "clickhouse.radondb.com/v1"
kind: "ClickHouseInstallation"
metadata:
  name: "pv-log"
spec:
  configuration:
    clusters:
      - name: "deployment-pv"
        # Templates are specified for this cluster explicitly
        templates:
          podTemplate: pod-template-with-volumes
        layout:
          shardsCount: 1
          replicasCount: 1

  templates:
    podTemplates:
      - name: pod-template-with-volumes
        spec:
          containers:
            - name: clickhouse
              image: yandex/clickhouse-server:19.3.7
              ports:
                - name: http
                  containerPort: 8123
                - name: client
                  containerPort: 9000
                - name: interserver
                  containerPort: 9009
              volumeMounts:
                - name: data-storage-vc-template
                  mountPath: /var/lib/clickhouse
                - name: log-storage-vc-template
                  mountPath: /var/log/clickhouse-server

    volumeClaimTemplates:
      - name: data-storage-vc-template
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 3Gi
      - name: log-storage-vc-template
        spec:
          accessModes:
            - ReadWriteOnce
          resources:
            requests:
              storage: 2Gi
```

### 示例 4：自定义 ClickHouse 配置

[自定义 ClickHouse 配置示例](../chi-examples/05-settings-01-overview.yaml) 定义了 Operator 可配置 RadonDB ClickHouse 集群。

```bash
$ kubectl apply -n test-clickhouse-operator -f https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/docs/chi-examples/05-settings-01-overview.yaml
clickhouseinstallation.clickhouse.radondb.com/settings-01 created
```

```yaml
apiVersion: "clickhouse.radondb.com/v1"
kind: "ClickHouseInstallation"
metadata:
  name: "settings-01"
spec:
  configuration:
    users:
      # test user has 'password' specified, while admin user has 'password_sha256_hex' specified
      test/password: qwerty
      test/networks/ip:
        - "127.0.0.1/32"
        - "192.168.74.1/24"
      test/profile: test_profile
      test/quota: test_quota
      test/allow_databases/database:
        - "dbname1"
        - "dbname2"
        - "dbname3"
      # admin use has 'password_sha256_hex' so actual password value is not published
      admin/password_sha256_hex: 8bd66e4932b4968ec111da24d7e42d399a05cb90bf96f587c3fa191c56c401f8
      admin/networks/ip: "127.0.0.1/32"
      admin/profile: default
      admin/quota: default
      # readonly user has 'password' field specified, not 'password_sha256_hex' as admin user above
      readonly/password: readonly_password
      readonly/profile: readonly
      readonly/quota: default
    profiles:
      test_profile/max_memory_usage: "1000000000"
      test_profile/readonly: "1"
      readonly/readonly: "1"
    quotas:
      test_quota/interval/duration: "3600"
    settings:
      compression/case/method: zstd
      disable_internal_dns_cache: 1
    files:
      dict1.xml: |
        <yandex>
            <!-- ref to file /etc/clickhouse-data/config.d/source1.csv -->
        </yandex>
      source1.csv: |
        a1,b1,c1,d1
        a2,b2,c2,d2
    clusters:
      - name: "standard"
        layout:
          shardsCount: 1
          replicasCount: 1
```

### 验证集群部署

集群部署成功后，您可以通过如下命令，验证集群运行状态，并查看服务详情。

```bash
$ kubectl get pods -n test-clickhouse-operator
NAME                    READY   STATUS    RESTARTS   AGE
chi-b3d29f-a242-0-0-0   1/1     Running   0          10m
```

```bash
$ kubectl get service -n test-clickhouse-operator
NAME                    TYPE           CLUSTER-IP       EXTERNAL-IP                          PORT(S)                         AGE
chi-b3d29f-a242-0-0     ClusterIP      None             <none>                               8123/TCP,9000/TCP,9009/TCP      11m
clickhouse-example-01   LoadBalancer   100.64.167.170   abc-123.us-east-1.elb.amazonaws.com   8123:30954/TCP,9000:32697/TCP   11m
```

## 访问 RadonDB ClickHouse

### 通过 EXTERNAL-IP 访问

执行 `kubectl get service -n <namespace>` 命令获取 **EXTERNAL-IP**，您可直接访问数据库。

```bash
$ clickhouse-client -h abc-123.us-east-1.elb.amazonaws.com -u clickhouse_operator --password clickhouse_operator_password 
ClickHouse client version 18.14.12.
Connecting to abc-123.us-east-1.elb.amazonaws.com:9000.
Connected to ClickHouse server version 19.4.3 revision 54416.
```

### 通过 pod-NAME 访问

若无可用 **EXTERNAL-IP**，您可在 Kubernetes 集群内，通过 **pod-NAME** 访问数据库。

```bash
$ kubectl -n test-clickhouse-operator exec -it chi-b3d29f-a242-0-0-0 -- clickhouse-client
ClickHouse client version 19.4.3.11.
Connecting to localhost:9000 as user default.
Connected to ClickHouse server version 19.4.3 revision 54416.
```
