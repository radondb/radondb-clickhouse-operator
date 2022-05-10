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

import (
	chiv1 "github.com/radondb/clickhouse-operator/pkg/apis/clickhouse.radondb.com/v1"
	"github.com/radondb/clickhouse-operator/pkg/model"
)

// ZooKeeperCanDeletePVC checks whether PVC on a host can be deleted
func ZooKeeperCanDeletePVC(chi *chiv1.ClickHouseInstallation, pvcName string) bool {
	// In any unknown cases just delete PVC with unclear bindings
	policy := chiv1.PVCReclaimPolicyDelete

	volumeClaimTemplate, ok := chi.GetZooKeeperVolumeClaimTemplate()

	if ok {
		for index := 0; index < int(chi.Spec.Configuration.Zookeeper.Replica); index++ {
			if pvcName == model.CreatePVCZooKeeperName(chi, index, volumeClaimTemplate) {
				policy = volumeClaimTemplate.PVCReclaimPolicy
				break
			}
		}
	}

	// Delete all explicitly specified as deletable PVCs and all PVCs of un-templated or unclear origin
	return policy == chiv1.PVCReclaimPolicyDelete
}
