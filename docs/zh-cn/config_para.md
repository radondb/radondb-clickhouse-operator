Contents
=================
- [Contents](#contents)
- [配置](#配置)
  - [简介](#简介)
    - [ClickHouse Cluster](#clickhouse-cluster)
    - [BusyBox](#busybox)
    - [ZooKeeper](#zookeeper)

# 配置

> [English](../en-us/config_para.md) | 简体中文

## 简介

RadonDB ClickHouse is an open-source, cloud-native, highly availability cluster solutions based on [ClickHouse](https://clickhouse.tech/). It provides features such as high availability, PB storage, real-time analytical, architectural stability and scalability.

### ClickHouse Cluster

|参数 |  描述 |  默认值 |
|:----|:----|:----|
|   `clickhouse.clusterName`   |  ClickHouse cluster name. | all-nodes  |
|   `clickhouse.shardscount`   |  Shards count. Once confirmed, it cannot be reduced.  |   1  |
|   `clickhouse.replicascount`   |  Replicas count. Once confirmed, it cannot be modified.  |   2  |
|   `clickhouse.image`   |  ClickHouse image name, it is not recommended to modify.  | radondb/clickhouse-server:v21.1.3.32-stable  |
|   `clickhouse.imagePullPolicy`   |  Image pull policy. The value can be Always/IfNotPresent/Never.  | IfNotPresent  |
|   `clickhouse.resources.memory`   |  K8s memory resources should be requested by a single Pod.  |  1Gi |
|   `clickhouse.resources.cpu`   |  K8s CPU resources should be requested by a single Pod.  |  0.5 |
|   `clickhouse.resources.storage`   |  K8s Storage resources should be requested by a single Pod.  |  10Gi  |
|   `clickhouse.user`   |  ClickHouse user array. Each user needs to contain a username, password and networks array.  | [{"username": "clickhouse", "password": "c1ickh0use0perator", "networks": ["127.0.0.1", "::/0"]}]  |
|   `clickhouse.port.tcp`   |  Port for the native interface.  |  9000  |
|   `clickhouse.port.http`   |  Port for HTTP/REST interface.  |  8123  |
|   `clickhouse.svc.type`   |  K8s service type. The value can be ClusterIP/NodePort/LoadBalancer.  |  ClusterIP  |
|   `clickhouse.svc.qceip`   |  If the value of type is LoadBalancer, You need to configure loadbalancer that provided by third-party platforms.     |  nil   |

### BusyBox

|参数 |  描述 |  默认值 |
|:----|:----|:----|
|   `busybox.image`   |  BusyBox image name, it is not recommended to modify.  |  busybox  |
|   `busybox.imagePullPolicy`   |  Image pull policy. The value can be Always/IfNotPresent/Never.  |  Always  |

### ZooKeeper

|参数 |  描述 |  默认值 |
|:----|:----|:----|
|   `zookeeper.install`   |  Whether to create ZooKeeper by operator.  |  true  |
|   `zookeeper.port`   |  ZooKeeper service port.   |  2181  |
|   `zookeeper.replicas`   |  ZooKeeper cluster replicas count.  |  3  |
|   `zookeeper.image`   |  ZooKeeper image name, it is not recommended to modify.  |  radondb/zookeeper:3.6.2  |
|   `zookeeper.imagePullPolicy`   |  Image pull policy. The value can be Always/IfNotPresent/Never.  |  Always  |
|   `zookeeper.resources.memory`   |  K8s memory resources should be requested by a single Pod.  | Deprecated, if install = true  |
|   `zookeeper.resources.cpu`   |  K8s CPU resources should be requested by a single Pod.  |  Deprecated, if install = true  |
|   `zookeeper.resources.storage`   |  K8s storage resources should be requested by a single Pod.  |  Deprecated, if install = true  |
