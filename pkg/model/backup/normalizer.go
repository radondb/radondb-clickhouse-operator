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

package backup

import (
	"context"
	"errors"
	chopmodel "github.com/radondb/clickhouse-operator/pkg/model"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "k8s.io/client-go/kubernetes"

	log "github.com/radondb/clickhouse-operator/pkg/announcer"
	chiV1 "github.com/radondb/clickhouse-operator/pkg/apis/clickhouse.radondb.com/v1"
	chopclientset "github.com/radondb/clickhouse-operator/pkg/client/clientset/versioned"
)

// Normalizer specifies structures normalizer
type Normalizer struct {
	kubeClient kube.Interface
	chb        *chiV1.ClickHouseBackup
	chopClient chopclientset.Interface
}

// NewNormalizer creates new normalizer
func NewNormalizer(kubeClient kube.Interface, chopClient chopclientset.Interface) *Normalizer {
	return &Normalizer{
		kubeClient: kubeClient,
		chopClient: chopClient,
	}
}

// CreateTemplatedCHB produces ready-to-use CHB object
func (n *Normalizer) CreateTemplatedCHB(chb *chiV1.ClickHouseBackup) (*chiV1.ClickHouseBackup, error) {
	if chb == nil {
		chb = new(chiV1.ClickHouseBackup)
	}

	n.chb = chb.DeepCopy()

	return n.normalize()
}

// normalize Returns normalized CHB
func (n *Normalizer) normalize() (*chiV1.ClickHouseBackup, error) {
	var err error

	// Walk over chbSpec datatype fields
	n.chb.Spec.CHIName = n.normalizeCHIName(n.chb.Spec.CHIName)
	n.chb.Spec.ClusterName = n.normalizeClusterName(n.chb.Spec.ClusterName)
	n.chb.Spec.Namespace = n.normalizeNamespace(n.chb.Spec.Namespace)
	n.chb.Spec.TCPPort = n.normalizeTCPPort(n.chb.Spec.TCPPort)
	n.chb.Spec.Image = n.normalizeImage(n.chb.Spec.Image)
	n.chb.Spec.ImagePullPolicy = n.normalizeImagePullPolicy(n.chb.Spec.ImagePullPolicy)
	n.chb.Spec.Backup = n.normalizeBackup(n.chb.Spec.Backup)
	n.chb.Spec.Restore = n.normalizeRestore(n.chb.Spec.Restore)

	cluster, err := n.normalizeCluster(n.chb.Cluster)

	if err != nil {
		return nil, err
	}

	n.chb.Cluster = cluster

	return n.chb, err
}

// normalizeCHIName normalizes .spec.chiName
func (n *Normalizer) normalizeCHIName(chiName string) string {
	return chiName
}

// normalizeClusterName normalizes .spec.clusterName
func (n *Normalizer) normalizeClusterName(clusterName string) string {
	return clusterName
}

// normalizeNamespace normalizes .spec.namespace
func (n *Normalizer) normalizeNamespace(namespace string) string {
	if namespace == "" {
		return v1.NamespaceDefault
	}

	return namespace
}

// normalizeTCPPort normalizes .spec.tcpPort
func (n *Normalizer) normalizeTCPPort(tcpPort int32) int32 {
	if tcpPort < 1 || tcpPort > 65535 {
		return chDefaultTCPPortNumber
	}
	return tcpPort
}

// normalizeImage normalizes .spec.image
func (n *Normalizer) normalizeImage(image string) string {
	if image == "" {
		return defaultClickHouseClientDockerImage
	}
	return image
}

// normalizeImagePullPolicy normalizes .spec.imagePullPolicy
func (n *Normalizer) normalizeImagePullPolicy(imagePullPolicy string) string {
	if strings.EqualFold(imagePullPolicy, imagePullPolicyPullNever) ||
		strings.EqualFold(imagePullPolicy, imagePullPolicyIfNotPresent) ||
		strings.EqualFold(imagePullPolicy, imagePullPolicyAlways) {
		// It is bool, use as it is
		return imagePullPolicy
	}

	return imagePullPolicyAlways
}

// normalizeBackup normalizes .spec.backupConfig
func (n *Normalizer) normalizeBackup(backup *chiV1.BackupConfig) *chiV1.BackupConfig {
	if backup.IsEmpty() {
		return backup
	}

	backup.Kind = n.normalizeBackupKind(backup.Kind)
	backup.Schedule = n.normalizeBackupSchedule(backup.Schedule)

	return backup
}

// normalizeBackupKind normalizes .spec.backupConfig.kind
func (n *Normalizer) normalizeBackupKind(kind string) string {
	if strings.EqualFold(kind, "") {
		return kind
	}

	if strings.EqualFold(kind, chiV1.ClickHouseBackupKindSingle) || strings.EqualFold(kind, chiV1.ClickHouseBackupKindSchedule) {
		return kind
	}

	return chiV1.ClickHouseBackupKindSingle
}

// normalizeBackupSchedule normalizes .spec.backupConfig.schedule
func (n *Normalizer) normalizeBackupSchedule(schedule string) string {
	// TODO

	return schedule
}

// normalizeRestore normalizes .spec.restoreConfig
func (n *Normalizer) normalizeRestore(restore *chiV1.RestoreConfig) *chiV1.RestoreConfig {
	if restore.IsEmpty() {
		return restore
	}

	restore.BackupName = n.normalizeRestoreBackupName(restore.BackupName)

	return restore
}

// normalizeRestoreBackupName normalizes .spec.restoreConfig.backupName
func (n *Normalizer) normalizeRestoreBackupName(backupName string) string {
	return backupName
}

// normalizeCluster normalizes .spec.cluster
func (n *Normalizer) normalizeCluster(cluster *chiV1.ChiCluster) (*chiV1.ChiCluster, error) {
	if cluster != nil && cluster.Name == n.chb.Spec.ClusterName && cluster.CHI.Name == n.chb.Spec.CHIName {
		return cluster, nil
	}

	log.V(1).M(n.chb).F().Info("namespace: %v, chiName: %v", n.chb.Spec.Namespace, n.chb.Spec.CHIName)

	chi, err := n.chopClient.ClickhouseV1().ClickHouseInstallations(n.chb.Spec.Namespace).Get(context.TODO(), n.chb.Spec.CHIName, metav1.GetOptions{})

	if err != nil {
		log.V(1).M(n.chb).F().Error("ERROR get chi %v:%v failed", n.chb.Spec.CHIName, n.chb.Spec.Namespace)
		return nil, err
	}

	normalizer := chopmodel.NewNormalizer(n.kubeClient)
	chi, err = normalizer.CreateTemplatedCHI(chi)
	if err != nil {
		log.V(1).M(n.chb).A().Error("FAILED to normalize CHI : %v", err)
	}

	log.V(1).M(n.chb).F().Info("chi: %v", chi.Spec.Configuration.Clusters[0])
	log.V(1).M(n.chb).F().Info("pods: %v", chi.Status.Pods)
	log.V(1).M(n.chb).F().Info("fqdns: %v", chi.Status.FQDNs)
	log.V(1).M(n.chb).F().Info("normalized: %v", chi.Status.NormalizedCHI.Configuration.Clusters[0])

	// check clusterName
	hasCluster := false
	for index := range chi.Spec.Configuration.Clusters {
		cluster = chi.Spec.Configuration.Clusters[index]
		log.V(1).M(n.chb).F().Info("%v : %v", cluster.Name, n.chb.Spec.ClusterName)
		if strings.EqualFold(cluster.Name, n.chb.Spec.ClusterName) {
			hasCluster = true
			break
		}
	}

	if !hasCluster {
		return nil, errors.New("cluster specified does not exist")
	}

	return cluster, nil
}
