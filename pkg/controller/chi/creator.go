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

package chi

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apps "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	"k8s.io/api/core/v1"

	log "github.com/radondb/clickhouse-operator/pkg/announcer"
	chiv1 "github.com/radondb/clickhouse-operator/pkg/apis/clickhouse.radondb.com/v1"
	"github.com/radondb/clickhouse-operator/pkg/chop"
	"github.com/radondb/clickhouse-operator/pkg/util"
	"github.com/radondb/clickhouse-operator/pkg/util/retry"
)

// createJob
func (c *Controller) createJob(ctx context.Context, job *batchv1.Job) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	log.V(1).Info("Create Job %s/%s", job.Namespace, job.Name)
	if _, err := c.kubeClient.BatchV1().Jobs(job.Namespace).Create(ctx, job, newCreateOptions()); err != nil {
		return err
	}

	return nil
}

// createCronJob
func (c *Controller) createCronJob(ctx context.Context, cronJob *batchv1beta1.CronJob) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	log.V(1).Info("Create CronJob %s/%s", cronJob.Namespace, cronJob.Name)
	if _, err := c.kubeClient.BatchV1beta1().CronJobs(cronJob.Namespace).Create(ctx, cronJob, newCreateOptions()); err != nil {
		return err
	}

	return nil
}

// createStatefulSetZooKeeper is an internal function, used in reconcileStatefulSet only
func (c *Controller) createStatefulSetZooKeeper(ctx context.Context, statefulSet *apps.StatefulSet, chi *chiv1.ClickHouseInstallation) error {
	log.V(1).M(chi).F().P()

	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	log.V(1).Info("Create StatefulSet %s/%s", statefulSet.Namespace, statefulSet.Name)
	if _, err := c.kubeClient.AppsV1().StatefulSets(statefulSet.Namespace).Create(ctx, statefulSet, newCreateOptions()); err != nil {
		// Unable to create StatefulSet at all
		return err
	}

	// StatefulSet created, wait until it is ready

	if err := c.waitZooKeeperReady(ctx, statefulSet); err == nil {
		// Target generation reached, StatefulSet created successfully
		return nil
	}

	// Unable to run StatefulSet, StatefulSet create failed, time to rollback?
	return c.onStatefulSetZooKeeperCreateFailed(ctx, statefulSet, chi)
}

// createStatefulSet is an internal function, used in reconcileStatefulSet only
func (c *Controller) createStatefulSet(ctx context.Context, statefulSet *apps.StatefulSet, host *chiv1.ChiHost) error {
	log.V(1).M(host).F().P()

	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	log.V(1).Info("Create StatefulSet %s/%s", statefulSet.Namespace, statefulSet.Name)
	if _, err := c.kubeClient.AppsV1().StatefulSets(statefulSet.Namespace).Create(ctx, statefulSet, newCreateOptions()); err != nil {
		// Unable to create StatefulSet at all
		return err
	}

	// StatefulSet created, wait until host is ready
	if err := c.waitHostReady(ctx, host); err == nil {
		// Target generation reached, StatefulSet created successfully
		return nil
	}

	// StatefulSet create failed, time to rollback?
	return c.onStatefulSetCreateFailed(ctx, statefulSet, host)
}

// updateStatefulSet is an internal function, used in reconcileStatefulSet only
func (c *Controller) updateStatefulSet(
	ctx context.Context,
	oldStatefulSet *apps.StatefulSet,
	newStatefulSet *apps.StatefulSet,
	host *chiv1.ChiHost,
) error {
	log.V(2).M(host).F().P()

	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil
	}

	// Apply newStatefulSet and wait for Generation to change
	updatedStatefulSet, err := c.kubeClient.AppsV1().StatefulSets(newStatefulSet.Namespace).Update(ctx, newStatefulSet, newUpdateOptions())
	if err != nil {
		// Update failed
		log.V(1).M(host).A().Error("%v", err)
		return err
	}

	// After calling "Update()"
	// 1. ObjectMeta.Generation is target generation
	// 2. Status.ObservedGeneration may be <= ObjectMeta.Generation

	if updatedStatefulSet.Generation == oldStatefulSet.Generation {
		// Generation is not updated - no changes in .spec section were made
		log.V(2).M(host).F().Info("no generation change")
		return nil
	}

	log.V(1).M(host).F().Info("generation change %d=>%d", oldStatefulSet.Generation, updatedStatefulSet.Generation)

	if err := c.waitHostReady(ctx, host); err == nil {
		// Target generation reached, StatefulSet updated successfully
		return nil
	}
	// Unable to run StatefulSet, StatefulSet update failed, time to rollback?
	return c.onStatefulSetUpdateFailed(ctx, oldStatefulSet, host)
}

// updateStatefulSet is an internal function, used in reconcileStatefulSet only
func (c *Controller) updatePersistentVolume(ctx context.Context, pv *v1.PersistentVolume) (*v1.PersistentVolume, error) {
	log.V(2).M(pv).F().P()
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil, fmt.Errorf("ctx is done")
	}

	var err error
	pv, err = c.kubeClient.CoreV1().PersistentVolumes().Update(ctx, pv, newUpdateOptions())
	if err != nil {
		// Update failed
		log.V(1).M(pv).A().Error("%v", err)
		return nil, err
	}

	return pv, err
}

func (c *Controller) updatePersistentVolumeClaim(ctx context.Context, pvc *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	log.V(2).M(pvc).F().P()
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return nil, fmt.Errorf("ctx is done")
	}
	old, getErr := c.kubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, newGetOptions())

	_, err := c.kubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(ctx, pvc, newUpdateOptions())
	if err != nil {
		// Update failed
		//if strings.Contains(err.Error(), "field can not be less than previous value") {
		//	return pvc, nil
		//} else {
		log.V(1).M(pvc).A().Error("unable to update PVC %v", err)
		//	return nil, err
		//}
	} else if getErr == nil {
		if old.Spec.Resources.Requests.Storage().Cmp(*pvc.Spec.Resources.Requests.Storage()) == -1 {
			c.waitUpdatePersistentVolumeClaimSuccess(ctx, pvc)
		}
	}

	return pvc, err
}

var errAbort = errors.New("onStatefulSetCreateFailed - abort")
var errStop = errors.New("onStatefulSetCreateFailed - stop")
var errIgnore = errors.New("onStatefulSetCreateFailed - ignore")
var errUnexpectedFlow = errors.New("unexpected flow")

// onStatefulSetCreateFailed handles situation when StatefulSet create failed
// It can just delete failed StatefulSet or do nothing
func (c *Controller) onStatefulSetCreateFailed(ctx context.Context, failedStatefulSet *apps.StatefulSet, host *chiv1.ChiHost) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return errIgnore
	}

	// What to do with StatefulSet - look into chop configuration settings
	switch chop.Config().OnStatefulSetCreateFailureAction {
	case chiv1.OnStatefulSetCreateFailureActionAbort:
		// Report appropriate error, it will break reconcile loop
		log.V(1).M(host).F().Info("abort")
		return errAbort

	case chiv1.OnStatefulSetCreateFailureActionDelete:
		// Delete gracefully failed StatefulSet
		log.V(1).M(host).F().Info("going to DELETE FAILED StatefulSet %s", util.NamespaceNameString(failedStatefulSet.ObjectMeta))
		_ = c.deleteHost(ctx, host)
		return c.shouldContinueOnCreateFailed()

	case chiv1.OnStatefulSetCreateFailureActionIgnore:
		// Ignore error, continue reconcile loop
		log.V(1).M(host).F().Info("going to ignore error %s", util.NamespaceNameString(failedStatefulSet.ObjectMeta))
		return errIgnore

	default:
		log.V(1).M(host).A().Error("Unknown c.chop.Config().OnStatefulSetCreateFailureAction=%s", chop.Config().OnStatefulSetCreateFailureAction)
		return errIgnore
	}

	return errUnexpectedFlow
}

// onStatefulSetZooKeeperCreateFailed handles situation when StatefulSet create failed
// It can just delete failed StatefulSet or do nothing
func (c *Controller) onStatefulSetZooKeeperCreateFailed(ctx context.Context, failedStatefulSet *apps.StatefulSet, chi *chiv1.ClickHouseInstallation) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return errIgnore
	}

	// What to do with StatefulSet - look into chop configuration settings
	switch chop.Config().OnStatefulSetCreateFailureAction {
	case chiv1.OnStatefulSetCreateFailureActionAbort:
		// Report appropriate error, it will break reconcile loop
		log.V(1).M(chi).F().Info("abort")
		return errAbort

	case chiv1.OnStatefulSetCreateFailureActionDelete:
		// Delete gracefully failed StatefulSet
		log.V(1).M(chi).F().Info("going to DELETE FAILED StatefulSet %s", util.NamespaceNameString(failedStatefulSet.ObjectMeta))
		_ = c.deleteZooKeeper(ctx, chi)
		return c.shouldContinueOnCreateFailed()

	case chiv1.OnStatefulSetCreateFailureActionIgnore:
		// Ignore error, continue reconcile loop
		log.V(1).M(chi).F().Info("going to ignore error %s", util.NamespaceNameString(failedStatefulSet.ObjectMeta))
		return errIgnore

	default:
		log.V(1).M(chi).A().Error("Unknown c.chop.Config().OnStatefulSetCreateFailureAction=%s", chop.Config().OnStatefulSetCreateFailureAction)
		return errIgnore
	}

	return errUnexpectedFlow
}

// onStatefulSetUpdateFailed handles situation when StatefulSet update failed
// It can try to revert StatefulSet to its previous version, specified in rollbackStatefulSet
func (c *Controller) onStatefulSetUpdateFailed(ctx context.Context, rollbackStatefulSet *apps.StatefulSet, host *chiv1.ChiHost) error {
	if util.IsContextDone(ctx) {
		log.V(2).Info("ctx is done")
		return errIgnore
	}

	// Convenience shortcuts
	namespace := rollbackStatefulSet.Namespace
	name := rollbackStatefulSet.Name

	// What to do with StatefulSet - look into chop configuration settings
	switch chop.Config().OnStatefulSetUpdateFailureAction {
	case chiv1.OnStatefulSetUpdateFailureActionAbort:
		// Report appropriate error, it will break reconcile loop
		log.V(1).M(host).F().Info("abort StatefulSet %s", util.NamespaceNameString(rollbackStatefulSet.ObjectMeta))
		return errAbort

	case chiv1.OnStatefulSetUpdateFailureActionRollback:
		// Need to revert current StatefulSet to oldStatefulSet
		log.V(1).M(host).F().Info("going to ROLLBACK FAILED StatefulSet %s", util.NamespaceNameString(rollbackStatefulSet.ObjectMeta))
		statefulSet, err := c.kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, name, newGetOptions())
		if err != nil {
			// Unable to fetch current StatefulSet
			return err
		}

		// Make copy of "previous" .Spec just to be sure nothing gets corrupted
		// Update StatefulSet to its 'previous' oldStatefulSet - this is expected to rollback inapplicable changes
		// Having StatefulSet .spec in rolled back status we need to delete current Pod - because in case of Pod being seriously broken,
		// it is the only way to go. Just delete Pod and StatefulSet will recreated Pod with current .spec
		// This will rollback Pod to previous .spec
		statefulSet.Spec = *rollbackStatefulSet.Spec.DeepCopy()
		statefulSet, _ = c.kubeClient.AppsV1().StatefulSets(namespace).Update(ctx, statefulSet, newUpdateOptions())
		_ = c.statefulSetDeletePod(ctx, statefulSet, host)

		return c.shouldContinueOnUpdateFailed()

	case chiv1.OnStatefulSetUpdateFailureActionIgnore:
		// Ignore error, continue reconcile loop
		log.V(1).M(host).F().Info("going to ignore error %s", util.NamespaceNameString(rollbackStatefulSet.ObjectMeta))
		return errIgnore

	default:
		log.V(1).M(host).A().Error("Unknown c.chop.Config().OnStatefulSetUpdateFailureAction=%s", chop.Config().OnStatefulSetUpdateFailureAction)
		return errIgnore
	}

	return errUnexpectedFlow
}

// shouldContinueOnCreateFailed return nil in case 'continue' or error in case 'do not continue'
func (c *Controller) shouldContinueOnCreateFailed() error {
	// Check configuration option regarding should we continue when errors met on the way
	// c.chopConfig.OnStatefulSetUpdateFailureAction
	var continueUpdate = false
	if continueUpdate {
		// Continue update
		return errIgnore
	}

	// Do not continue update
	return errStop
}

// shouldContinueOnUpdateFailed return nil in case 'continue' or error in case 'do not continue'
func (c *Controller) shouldContinueOnUpdateFailed() error {
	// Check configuration option regarding should we continue when errors met on the way
	// c.chopConfig.OnStatefulSetUpdateFailureAction
	var continueUpdate = false
	if continueUpdate {
		// Continue update
		return errIgnore
	}

	// Do not continue update
	return errStop
}

// waitUpdatePersistentVolumeClaimSuccess wait pvc's .status.conditions.status to `FileSystemResizePending`.
func (c *Controller) waitUpdatePersistentVolumeClaimSuccess(ctx context.Context, pvc *v1.PersistentVolumeClaim) {
	// get persistentVolumeClaim and judge if it is provided by qingcloud.
	currentPVC, err := c.kubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, newGetOptions())
	if err != nil {
		log.A().Error("unable to get PersistentVolumeClaims(%s/%s) err: %v", pvc.Namespace, pvc.Name, err)
		return
	}

	provisioner, ok := currentPVC.ObjectMeta.Annotations["volume.beta.kubernetes.io/storage-provisioner"]
	log.V(1).M(pvc).Info("get PersistentVolumeClaims(%s/%s) - provisioner: %s", pvc.Namespace, pvc.Name, provisioner)
	if ok && strings.Contains(provisioner, "qingcloud") {
		// provided by qingcloud, wait status changes to FileSystemResizePending.
		maxTries := 10
		err := retry.Retry(ctx, maxTries, "check PVC's status", log.A(), func() error {
			currentPVC, err := c.kubeClient.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, newGetOptions())
			if err != nil {
				log.A().Error("unable to get PersistentVolumeClaims(%s/%s) err: %v", pvc.Namespace, pvc.Name, err)
				return err
			}

			conditions := currentPVC.Status.Conditions
			if conditions == nil {
				return errors.New("no conditions")
			}

			status := conditions[0].Type
			if status != "FileSystemResizePending" {
				return errors.New(string("not yet `FileSystemResizePending` status, status = " + status))
			}

			return nil
		},
		)
		if err != nil {
			log.A().Error("Resize PersistentVolumeClaims(%s/%s) - failed with error: %v", pvc.Namespace, pvc.Name, err)
		}
	}
}
