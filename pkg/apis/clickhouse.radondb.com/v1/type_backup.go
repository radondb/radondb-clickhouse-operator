// Copyright 2020 [RadonDB](https://github.com/radondb). All rights reserved.
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

package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClickHouseBackup struct {
	metav1.TypeMeta   `json:",inline"               yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"    yaml:"metadata,omitempty"`
	Spec              ClickHouseBackupSpec `json:"spec"                  yaml:"spec"`
	ChbStatus         ChbStatus            `json:"status"                yaml:"status"`
}

type ClickHouseBackupSpec struct {
	CHIName     string         `json:"chiName,omitempty"          yaml:"chiName,omitempty"`
	ClusterName string         `json:"clusterName,omitempty"      yaml:"clusterName,omitempty"`
	Namespace   string         `json:"namespace,omitempty"        yaml:"namespace,omitempty"`
	Backup      *BackupConfig  `json:"backup,omitempty"           yaml:"backup,omitempty"`
	Restore     *RestoreConfig `json:"restore,omitempty"          yaml:"restore,omitempty"`
}

type BackupConfig struct {
	Kind     string `json:"kind,omitempty"     yaml:"kind,omitempty"`
	Schedule string `json:"schedule,omitempty" yaml:"schedule,omitempty"`
}

type RestoreConfig struct {
	BackupName string `json:"backupName,omitempty" yaml:"backupName,omitempty"`
}

type ChbStatus struct {
	Status  string      `json:"status"                   yaml:"status"`
	BackupName string   `json:"backupName"               yaml:"backupName"`
	Schedule   string   `json:"schedule"                 yaml:"schedule"`
	StartTime  string   `json:"startTime"                yaml:"startTime"`
	EndTime    string   `json:"endTime"                  yaml:"endTime"`
	BackupSize string   `json:"backupSize"               yaml:"backupSize"`
	Cluster *ChiCluster `json:"chiCluster,omitempty"     yaml:"chiCluster,omitempty"`
}

// NewClickHouseBackup create new ClickHouseBackup object
func NewClickHouseBackup() *ClickHouseBackup {
	return new(ClickHouseBackup)
}

func (chb *ClickHouseBackup) IsEmpty() bool {
	if chb == nil {
		return true
	}
	return false
}

func (chb *ClickHouseBackup) WalkHost(f func(host *ChiHost, shardIndex, replicaIndex int) error) error {
	for shardIndex := range chb.ChbStatus.Cluster.Layout.Shards {
		shard := &chb.ChbStatus.Cluster.Layout.Shards[shardIndex]
		for replicaIndex := range shard.Hosts {
			host := shard.Hosts[replicaIndex]

			if err := f(host, shardIndex, replicaIndex); err != nil {
				return err
			}
		}
	}

	return nil
}

// NewClickHouseBackupSpec create new ClickHouseBackupSpec object
func NewClickHouseBackupSpec() *ClickHouseBackupSpec {
	return new(ClickHouseBackupSpec)
}

// NewBackupConfig create new BackupConfig object
func NewBackupConfig() *BackupConfig {
	return new(BackupConfig)
}

func (backup *BackupConfig) IsEmpty() bool {
	return backup == nil || backup.Kind == ""
}

func (backup *BackupConfig) IsSingleBackup() bool {
	return backup != nil && backup.Kind == ClickHouseBackupKindSingle
}

func (backup *BackupConfig) IsScheduleBackup() bool {
	return backup != nil && backup.Kind == ClickHouseBackupKindSchedule
}

// NewRestoreConfig create new RestoreConfig object
func NewRestoreConfig() *RestoreConfig {
	return new(RestoreConfig)
}

func (restore *RestoreConfig) IsEmpty() bool {
	return restore == nil || restore.BackupName == ""
}

func (s *ChbStatus) ReconcileBackupRunning(time string) {
	s.Status = StatusBackupRunning
	s.StartTime = time
	s.EndTime = ""
	s.BackupName = ""
	s.BackupSize = ""
}

func (s *ChbStatus) ReconcileBackupCompleted(backupName string, schedule string, time string, size string) {
	s.Status = StatusBackupCompleted
	s.BackupName = backupName
	s.Schedule = schedule
	s.EndTime = time
	s.BackupSize = size
}

func (s *ChbStatus) ReconcileBackupFailed(time string) {
	s.Status = StatusBackupFailed
	s.BackupName = ""
	s.EndTime = time
	s.BackupSize = ""
}

func (s *ChbStatus) ReconcileBackupUnknow(time string) {
	s.Status = StatusBackupUnknow
	s.EndTime = time
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClickHouseBackupList defines a list of ClickHouseBackup resources
type ClickHouseBackupList struct {
	metav1.TypeMeta `json:",inline"  yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ClickHouseBackup `json:"items" yaml:"items"`
}
