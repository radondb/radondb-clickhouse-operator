Contents
=================
- [Contents](#contents)
- [Deploy Radondb ClickHouse On Kubernetes](#deploy-radondb-clickhouse-on-kubernetes)
  - [Introduction](#introduction)
  - [Prerequisites](#prerequisites)
  - [Procedure](#procedure)
    - [Step 1 : Add Helm Repository](#step-1--add-helm-repository)
    - [Step 2 : Install RadonDB ClickHouse Operator](#step-2--install-radondb-clickhouse-operator)
    - [Step 3 : Install RadonDB ClickHouse Cluster](#step-3--install-radondb-clickhouse-cluster)
    - [Step 4 : Verification](#step-4--verification)
      - [Check the Status of Pod](#check-the-status-of-pod)
      - [Check the Status of SVC](#check-the-status-of-svc)
  - [Access RadonDB ClickHouse](#access-radondb-clickhouse)
    - [Use Pod](#use-pod)
    - [Use Service](#use-service)
  - [Persistence](#persistence)
  - [Custom Configuration](#custom-configuration)

# Deploy Radondb ClickHouse On Kubernetes

> English | [简体中文](../zh-cn/deploy_radondb-clickhouse_with_operator_on_kubernetes.md)

## Introduction

RadonDB ClickHouse is an open-source, cloud-native, highly availability cluster solutions based on [ClickHouse](https://clickhouse.tech/) and [RadonDB ClickHouse Operator](https://github.com/radondb/radondb-clickhouse-operator). It provides features such as high availability, PB storage, real-time analytical, architectural stability and scalability.

This tutorial demonstrates how to deploy RadonDB ClickHouse on Kubernetes.

## Prerequisites

- You have created a Kubernetes cluster.
- You have installed helm.

## Procedure

### Step 1 : Add Helm Repository

Add and update helm repository.

```bash
helm repo add <repoName> https://radondb.github.io/radondb-clickhouse-kubernetes/
helm repo update
```

**Expected output**

```bash
$ helm repo add ck https://radondb.github.io/radondb-clickhouse-kubernetes/
"ck" has been added to your repositories

$ helm repo update
Hang tight while we grab the latest from your chart repositories...
...Successfully got an update from the "ck" chart repository
Update Complete. ⎈Happy Helming!⎈
```

### Step 2 : Install RadonDB ClickHouse Operator

```bash
helm install --generate-name -n <nameSpace> <repoName>/clickhouse-operator
```

**Expected output**

```bash
$ helm install clickhouse-operator ck/clickhouse-operator -n kube-system
NAME: clickhouse-operator
LAST DEPLOYED: Wed Aug 17 14:43:44 2021
NAMESPACE: kube-system
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

> **Notice**
> 
> This command will install ClickHouse Operator in the namespace `kube-system`. Therefore, ClickHouse Operator only needs to be installed once in a Kubernetes cluster.

### Step 3 : Install RadonDB ClickHouse Cluster

```bash
helm install --generate-name <repoName>/clickhouse-cluster -n <nameSpace>\
  --set <para_name>=<para_value>
```

- For more information about cluter parameters, see [Configuration](#configuration).
- If you need to customize many parameters, you can modify `values.yaml` file. For details, see [Custom Configuration](#custom-configuration).

**Expected output**

```bash
$ helm install clickhouse ck/clickhouse-cluster -n test
NAME: clickhouse
LAST DEPLOYED: Wed Aug 17 14:48:12 2021
NAMESPACE: test
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

### Step 4 : Verification

#### Check the Status of Pod

```bash
kubectl get pods -n <nameSpace>
```

**Expected output**

```bash
$ kubectl get pods -n test
NAME                                READY   STATUS    RESTARTS   AGE
pod/chi-ClickHouse-replicas-0-0-0   2/2     Running   0          3m13s
pod/chi-ClickHouse-replicas-0-1-0   2/2     Running   0          2m51s
pod/zk-clickhouse-cluster-0         1/1     Running   0          3m13s
pod/zk-clickhouse-cluster-1         1/1     Running   0          3m13s
pod/zk-clickhouse-cluster-2         1/1     Running   0          3m13s
```

#### Check the Status of SVC

```bash
kubectl get service -n <nameSpace>
```

**Expected output**

```bash
$ kubectl get service -n test
NAME                                  TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)                         AGE
service/chi-ClickHouse-replicas-0-0   ClusterIP   None            <none>        8123/TCP,9000/TCP,9009/TCP      2m53s
service/chi-ClickHouse-replicas-0-1   ClusterIP   None            <none>        8123/TCP,9000/TCP,9009/TCP      2m36s
service/clickhouse-ClickHouse         ClusterIP   10.96.137.152   <none>        8123/TCP,9000/TCP               3m14s
service/zk-client-clickhouse-cluster  ClusterIP   10.107.33.51    <none>        2181/TCP,7000/TCP               3m13s
service/zk-server-clickhouse-cluster  ClusterIP   None            <none>        2888/TCP,3888/TCP               3m13s
```

## Access RadonDB ClickHouse

### Use Pod

You can directly connect to ClickHouse Pod with `kubectl`.

```bash
kubectl exec -it <podName> -n <nameSpace> -- clickhouse-client --user=<userName> --password=<userPassword>
```

**Expected output**

```bash
$ kubectl get pods | grep clickhouse
chi-ClickHouse-replicas-0-0-0   1/1     Running   0          8m50s
chi-ClickHouse-replicas-0-1-0   1/1     Running   0          8m50s

$ kubectl exec -it chi-ClickHouse-replicas-0-0-0 -- clickhouse-client -u clickhouse --password=c1ickh0use0perator --query='select hostName()'
chi-ClickHouse-replicas-0-0-0
```

### Use Service

```sql
echo '<query>' | curl 'http://<username>:<password>@<svcIP>:<HTTPPort>/' --data-binary @-
```

**Expected output**

```bash
$ kubectl get service |grep clickhouse
clickhouse-ClickHouse            ClusterIP   10.96.137.152   <none>        9000/TCP,8123/TCP   12m
chi-ClickHouse-replicas-0-0      ClusterIP   None            <none>        9000/TCP,8123/TCP   12m
chi-ClickHouse-replicas-0-1      ClusterIP   None            <none>        9000/TCP,8123/TCP   12m
```

```sql
echo 'select hostname()' | curl 'http://clickhouse:c1ickh0use0perator@10.96.137.152:8123/' --data-binary @-
chi-ClickHouse-replicas-0-1-0
echo 'select hostname()' | curl 'http://clickhouse:c1ickh0use0perator@10.96.137.152:8123/' --data-binary @-
chi-ClickHouse-replicas-0-0-0
```

## Persistence

You can configure a Pod to use a PersistentVolumeClaim(PVC) for storage. 
In default, PVC mount on the `/var/lib/clickhouse` directory.

1. You should create a Pod that uses the above PVC for storage.

2. You should create a PVC that is automatically bound to a suitable PersistentVolume(PV). 

> **Notices**
> 
> PVC can use different PV, so using the different PV show the different performance.

## Custom Configuration

If you need to customize many parameters, you can modify [values.yaml](./helm-charts/clickhouse/clickhouse-with-clickhouse-operator/values.yaml).

1. Download the `values.yaml` file.
2. Modify the parameter values in the `values.yaml`.
3. Run the following command to deploy the cluster.

```bash
helm install --generate-name <repoName>/clickhouse-cluster -n <nameSpace>\
  -f /<path>/to/values.yaml
```
