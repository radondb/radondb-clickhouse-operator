// Copyright 2019 Altinity Ltd and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zookeeper

const (
	// dirPathZooKeeperData  specifies full path of data folder where ClickHouse would place its log files
	dirPathZooKeeperData = "/var/lib/zookeeper"
)

const (
	// zooKeeperContainerName Name of container within Pod with ZooKeeper instance.
	zooKeeperContainerName = "zookeeper"

	// Default ZooKeeper docker image to be used
	defaultZooKeeperDockerImage = "radondb/zookeeper"
)

const (
	defaultPrometheusPortName   = "prometheus"
	defaultPrometheusPortNumber = 7000
)

const (
	// ZooKeeper open ports
	zkDefaultClientPortName           = "client"
	zkDefaultClientPortNumber         = int32(2181)
	zkDefaultServerPortName           = "server"
	zkDefaultServerPortNumber         = int32(2888)
	zkDefaultLeaderElectionPortName   = "leader-election"
	zkDefaultLeaderElectionPortNumber = int32(3888)
)
