[English](docs/quick_start.md) | 简体中文 

---

- [Quick Start Guides](#quick-start-guides)
  - [Prerequisites](#prerequisites)
  - [RadonDB ClickHouse Operator Installation](#radondb-clickhouse-operator-installation)
    - [Case 1: Install operator into `kube-system` namespace](#case-1-install-operator-into-kube-system-namespace)
    - [Case 2: Install operator on kubernetes version prior `1.17` in `kube-system` namespace](#case-2-install-operator-on-kubernetes-version-prior-117-in-kube-system-namespace)
    - [Case 3: Customize installation parameters](#case-3-customize-installation-parameters)
    - [Case 4: Not run scripts from internet in your protected environment](#case-4-not-run-scripts-from-internet-in-your-protected-environment)
    - [Operator installation process](#operator-installation-process)
    - [Building ClickHouse Operator from Sources](#building-clickhouse-operator-from-sources)
  - [adonDB ClickHouse Cluster Installation](#adondb-clickhouse-cluster-installation)
    - [Create Custom Namespace](#create-custom-namespace)
    - [Example 1: Trivial](#example-1-trivial)
    - [Example 2: Simple Persistent Volume](#example-2-simple-persistent-volume)
  - [Example 3: Custom Deployment with Pod and VolumeClaim](#example-3-custom-deployment-with-pod-and-volumeclaim)
    - [Example 4: Custom Deployment with Specific ClickHouse Configuration](#example-4-custom-deployment-with-specific-clickhouse-configuration)
    - [Validate cluster deployment](#validate-cluster-deployment)
  - [Access to RadonDB ClickHouse](#access-to-radondb-clickhouse)
    - [Via EXTERNAL-IP](#via-external-ip)
    - [Via pod-NAME](#via-pod-name)

# Quick Start Guides

## Prerequisites

- Operational Kubernetes instance
- Properly configured `kubectl`
- `curl`

## RadonDB ClickHouse Operator Installation

Apply `clickhouse-operator` installation manifest. The simplest way - directly from `github`.

### Case 1: Install operator into `kube-system` namespace

just run:

```bash
kubectl apply -f https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/deploy/operator/clickhouse-operator-install-bundle.yaml
```

### Case 2: Install operator on kubernetes version prior `1.17` in `kube-system` namespace

just run:

```bash
kubectl apply -f https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/deploy/operator/clickhouse-operator-install-bundle-v1beta1.yaml
```

### Case 3: Customize installation parameters

such as namespace where to install operator or operator's image, use the special installer script.

```bash
curl -s https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/deploy/operator-web-installer/clickhouse-operator-install.sh | OPERATOR_NAMESPACE=test-clickhouse-operator bash
```

Take into account explicitly specified namespace.

This namespace would be created and used to install `clickhouse-operator` into.
Install script would download some `.yaml` and `.xml` files and install `clickhouse-operator` into specified namespace.

After installation **clickhouse-operator** will watch custom resources like a `kind: ClickhouseInstallation` only in `test-clickhouse-operator` namespace.

If no `OPERATOR_NAMESPACE` specified, as:

```bash
cd ~
curl -s https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/deploy/operator-web-installer/clickhouse-operator-install.sh | bash
```

Installer will install **clickhouse-operator** into `kube-system` namespace and will watch custom resources like a `kind: ClickhouseInstallation` in all available namespaces.

### Case 4: Not run scripts from internet in your protected environment

You can download manually [this template file](../deploy/operator/clickhouse-operator-install-template.yaml).

And edit it according to your choice. After that apply it with `kubectl`. Or you can use this snippet instead:

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

### Operator installation process

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

Check `clickhouse-operator` is running:

```bash
$ kubectl get pods -n test-clickhouse-operator
NAME                                 READY   STATUS    RESTARTS   AGE
clickhouse-operator-5ddc6d858f-drppt 1/1     Running   0          1m
```

### Building ClickHouse Operator from Sources

Complete instructions on how to build ClickHouse operator from sources as well as how to build a docker image and use it inside `kubernetes` described [here](../operator_build_from_sources.md).

## adonDB ClickHouse Cluster Installation

There are several ready-to-use [ClickHouseInstallation examples](../chi-examples/). Below are few ones to start with.

### Create Custom Namespace
It is a good practice to have all components run in dedicated namespaces. Let's run examples in `test` namespace
```bash
kubectl create namespace test-clickhouse-operator
```
```text
namespace/test-clickhouse-operator created
```

### Example 1: Trivial

This is the trivial [1 shard 1 replica](../chi-examples/01-simple-layout-01-1shard-1repl.yaml) example.

> **WARNING**
> 
> Do not use it for anything other than 'Hello, world!', it does not have persistent storage!
 
```bash
$ kubectl apply -n test-clickhouse-operator -f https://raw.githubusercontent.com/radondb/radondb-clickhouse-operator/master/docs/chi-examples/01-simple-layout-01-1shard-1repl.yaml
clickhouseinstallation.clickhouse.radondb.com/simple-01 created
```

Installation specification is straightforward and defines 1-replica cluster:

```yaml
apiVersion: "clickhouse.radondb.com/v1"
kind: "ClickHouseInstallation"
metadata:
  name: "simple-01"
```

### Example 2: Simple Persistent Volume

In case of having Dynamic Volume Provisioning available, we are able to use PersistentVolumeClaims.

Manifest is [available in examples](../chi-examples/03-persistent-volume-01-default-volume.yaml):

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

## Example 3: Custom Deployment with Pod and VolumeClaim

Let's install more complex example with:

- Deployment specified
- Pod template
- VolumeClaim template

Manifest is [available in examples](../chi-examples/03-persistent-volume-02-volume.yaml):

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

### Example 4: Custom Deployment with Specific ClickHouse Configuration

You can tell operator to configure your ClickHouse, as shown in the example below ([link to the manifest](../chi-examples/05-settings-01-overview.yaml):

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

### Validate cluster deployment

Once cluster is created, there are two checks to be made.

```bash
$ kubectl get pods -n test-clickhouse-operator
NAME                    READY   STATUS    RESTARTS   AGE
chi-b3d29f-a242-0-0-0   1/1     Running   0          10m
```

Watch out for 'Running' status. Also check services created by an operator:

```bash
$ kubectl get service -n test-clickhouse-operator
NAME                    TYPE           CLUSTER-IP       EXTERNAL-IP                          PORT(S)                         AGE
chi-b3d29f-a242-0-0     ClusterIP      None             <none>                               8123/TCP,9000/TCP,9009/TCP      11m
clickhouse-example-01   LoadBalancer   100.64.167.170   abc-123.us-east-1.elb.amazonaws.com   8123:30954/TCP,9000:32697/TCP   11m
```

ClickHouse is up and running!

## Access to RadonDB ClickHouse

### Via EXTERNAL-IP

In case previous command `kubectl get service -n test-clickhouse-operator` reported **EXTERNAL-IP**, we can directly access ClickHouse with:

```bash
$ clickhouse-client -h abc-123.us-east-1.elb.amazonaws.com -u clickhouse_operator --password clickhouse_operator_password 
ClickHouse client version 18.14.12.
Connecting to abc-123.us-east-1.elb.amazonaws.com:9000.
Connected to ClickHouse server version 19.4.3 revision 54416.
```

### Via pod-NAME

In case there is not **EXTERNAL-IP** available, we can access ClickHouse from inside Kubernetes cluster.

```bash
$ kubectl -n test-clickhouse-operator exec -it chi-b3d29f-a242-0-0-0 -- clickhouse-client
ClickHouse client version 19.4.3.11.
Connecting to localhost:9000 as user default.
Connected to ClickHouse server version 19.4.3 revision 54416.
```
