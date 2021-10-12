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

package model

import (
	"bytes"
	"fmt"
	"strconv"
	// "net/url"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	log "github.com/TCeason/clickhouse-operator/pkg/announcer"
	chiv1 "github.com/TCeason/clickhouse-operator/pkg/apis/clickhouse.radondb.com/v1"
	"github.com/TCeason/clickhouse-operator/pkg/chop"
	"github.com/TCeason/clickhouse-operator/pkg/util"
)

type Creator struct {
	chop                   *chop.CHOp
	chi                    *chiv1.ClickHouseInstallation
	chConfigFilesGenerator *ClickHouseConfigFilesGenerator
	labeler                *Labeler
	a                      log.Announcer
}

func NewCreator(chop *chop.CHOp, chi *chiv1.ClickHouseInstallation) *Creator {
	return &Creator{
		chop:                   chop,
		chi:                    chi,
		chConfigFilesGenerator: NewClickHouseConfigFilesGenerator(NewClickHouseConfigGenerator(chi), chop.Config()),
		labeler:                NewLabeler(chop, chi),
		a:                      log.M(chi),
	}
}

// CreateServiceCHI creates new corev1.Service for specified CHI
func (c *Creator) CreateServiceCHI() *corev1.Service {
	serviceName := CreateCHIServiceName(c.chi)

	c.a.V(1).F().Info("%s/%s", c.chi.Namespace, serviceName)
	if template, ok := c.chi.GetCHIServiceTemplate(); ok {
		// .templates.ServiceTemplate specified
		return c.createServiceFromTemplate(
			template,
			c.chi.Namespace,
			serviceName,
			c.labeler.getLabelsServiceCHI(),
			c.labeler.getSelectorCHIScopeReady(),
		)
	} else {
		// Incorrect/unknown .templates.ServiceTemplate specified
		// Create default Service
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: c.chi.Namespace,
				Labels:    c.labeler.getLabelsServiceCHI(),
			},
			Spec: corev1.ServiceSpec{
				// ClusterIP: templateDefaultsServiceClusterIP,
				Ports: []corev1.ServicePort{
					{
						Name:       chDefaultHTTPPortName,
						Protocol:   corev1.ProtocolTCP,
						Port:       chDefaultHTTPPortNumber,
						TargetPort: intstr.FromString(chDefaultHTTPPortName),
					},
					{
						Name:       chDefaultTCPPortName,
						Protocol:   corev1.ProtocolTCP,
						Port:       chDefaultTCPPortNumber,
						TargetPort: intstr.FromString(chDefaultTCPPortName),
					},
				},
				Selector: c.labeler.getSelectorCHIScopeReady(),
				Type:     corev1.ServiceTypeClusterIP,
			},
		}
	}
}

// CreateServiceZooKeeperServer creates new corev1.Service for zookeeper
func (c *Creator) CreateServiceZooKeeperServer(chi *chiv1.ClickHouseInstallation) *corev1.Service {
	zooKeeperStatefulSetName := CreateStatefulSetZooKeeperName(chi)
	zooKeeperServiceName := CreateStatefulSetServiceZooKeeperServerName(chi)

	c.a.V(1).F().Info("%s/%s for ZooKeeper Set %s", chi.Namespace, zooKeeperServiceName, zooKeeperStatefulSetName)

	// TODO: use template if exists.
	// if template, ok := host.GetServiceZooKeeperServerTemplate(); ok {
	//     return c.createServiceZooKeeperServerFromTemplate()
	// }

	// Create default Service
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zooKeeperServiceName,
			Namespace: chi.Namespace,
			Labels:    c.labeler.getLabelsServiceZooKeeperServer(),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       zkDefaultServerPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       zkDefaultServerPortNumber,
					TargetPort: intstr.FromInt(int(zkDefaultServerPortNumber)),
				},
				{
					Name:       zkDefaultLeaderElectionPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       zkDefaultLeaderElectionPortNumber,
					TargetPort: intstr.FromInt(int(zkDefaultLeaderElectionPortNumber)),
				},
			},
			Selector:  c.labeler.getSelectorZooKeeperScope(),
			ClusterIP: templateDefaultsServiceClusterIP,
		},
	}
}

// CreateServiceZooKeeperClient creates new corev1.Service for zookeeper
func (c *Creator) CreateServiceZooKeeperClient(chi *chiv1.ClickHouseInstallation) *corev1.Service {
	zooKeeperStatefulSetName := CreateStatefulSetZooKeeperName(chi)
	zooKeeperServiceName := CreateStatefulSetServiceZooKeeperClientName(chi)

	c.a.V(1).F().Info("%s/%s for ZooKeeper Set %s", chi.Namespace, zooKeeperServiceName, zooKeeperStatefulSetName)

	// TODO: use template if exists.
	// if template, ok := host.GetServiceZooKeeperClientTemplate(); ok {
	//     return c.createServiceZooKeeperClientFromTemplate()
	// }

	// Create default Service
	zookeeperConfig := chi.Spec.Configuration.Zookeeper
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zooKeeperServiceName,
			Namespace: chi.Namespace,
			Labels:    c.labeler.getLabelsServiceZooKeeperClient(),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       zkDefaultClientPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       zookeeperConfig.Port,
					TargetPort: intstr.FromInt(int(zookeeperConfig.Port)),
				},
			},
			Selector: c.labeler.getSelectorZooKeeperScope(),
		},
	}
}

// CreateServiceCluster creates new corev1.Service for specified Cluster
func (c *Creator) CreateServiceCluster(cluster *chiv1.ChiCluster) *corev1.Service {
	serviceName := CreateClusterServiceName(cluster)

	c.a.V(1).F().Info("%s/%s", cluster.Address.Namespace, serviceName)
	if template, ok := cluster.GetServiceTemplate(); ok {
		// .templates.ServiceTemplate specified
		return c.createServiceFromTemplate(
			template,
			cluster.Address.Namespace,
			serviceName,
			c.labeler.getLabelsServiceCluster(cluster),
			c.labeler.getSelectorClusterScopeReady(cluster),
		)
	} else {
		return nil
	}
}

// CreateServiceShard creates new corev1.Service for specified Shard
func (c *Creator) CreateServiceShard(shard *chiv1.ChiShard) *corev1.Service {
	serviceName := CreateShardServiceName(shard)

	c.a.V(1).F().Info("%s/%s", shard.Address.Namespace, serviceName)
	if template, ok := shard.GetServiceTemplate(); ok {
		// .templates.ServiceTemplate specified
		return c.createServiceFromTemplate(
			template,
			shard.Address.Namespace,
			serviceName,
			c.labeler.getLabelsServiceShard(shard),
			c.labeler.getSelectorShardScopeReady(shard),
		)
	} else {
		return nil
	}
}

// CreateServiceHost creates new corev1.Service for specified host
func (c *Creator) CreateServiceHost(host *chiv1.ChiHost) *corev1.Service {
	serviceName := CreateStatefulSetServiceName(host)
	statefulSetName := CreateStatefulSetName(host)

	c.a.V(1).F().Info("%s/%s for Set %s", host.Address.Namespace, serviceName, statefulSetName)
	if template, ok := host.GetServiceTemplate(); ok {
		// .templates.ServiceTemplate specified
		return c.createServiceFromTemplate(
			template,
			host.Address.Namespace,
			serviceName,
			c.labeler.getLabelsServiceHost(host),
			c.labeler.GetSelectorHostScope(host),
		)
	} else {
		// Incorrect/unknown .templates.ServiceTemplate specified
		// Create default Service
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: host.Address.Namespace,
				Labels:    c.labeler.getLabelsServiceHost(host),
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Name:       chDefaultHTTPPortName,
						Protocol:   corev1.ProtocolTCP,
						Port:       host.HTTPPort,
						TargetPort: intstr.FromInt(int(host.HTTPPort)),
					},
					{
						Name:       chDefaultTCPPortName,
						Protocol:   corev1.ProtocolTCP,
						Port:       host.TCPPort,
						TargetPort: intstr.FromInt(int(host.TCPPort)),
					},
					{
						Name:       chDefaultInterserverHTTPPortName,
						Protocol:   corev1.ProtocolTCP,
						Port:       host.InterserverHTTPPort,
						TargetPort: intstr.FromInt(int(host.InterserverHTTPPort)),
					},
				},
				Selector:                 c.labeler.GetSelectorHostScope(host),
				ClusterIP:                templateDefaultsServiceClusterIP,
				Type:                     "ClusterIP",
				PublishNotReadyAddresses: true,
			},
		}
	}
}

// verifyServiceTemplatePorts verifies ChiServiceTemplate to have reasonable ports specified
func (c *Creator) verifyServiceTemplatePorts(template *chiv1.ChiServiceTemplate) error {
	for i := range template.Spec.Ports {
		servicePort := &template.Spec.Ports[i]
		if (servicePort.Port < 1) || (servicePort.Port > 65535) {
			msg := fmt.Sprintf("template:%s INCORRECT PORT:%d", template.Name, servicePort.Port)
			c.a.V(1).F().Warning(msg)
			return fmt.Errorf(msg)
		}
	}
	return nil
}

// createServiceFromTemplate create Service from ChiServiceTemplate and additional info
func (c *Creator) createServiceFromTemplate(
	template *chiv1.ChiServiceTemplate,
	namespace string,
	name string,
	labels map[string]string,
	selector map[string]string,
) *corev1.Service {

	// Verify Ports
	if err := c.verifyServiceTemplatePorts(template); err != nil {
		return nil
	}

	// Create Service
	service := &corev1.Service{
		ObjectMeta: *template.ObjectMeta.DeepCopy(),
		Spec:       *template.Spec.DeepCopy(),
	}

	// Overwrite .name and .namespace - they are not allowed to be specified in template
	service.Name = name
	service.Namespace = namespace

	// Append provided Labels to already specified Labels in template
	service.Labels = util.MergeStringMapsOverwrite(service.Labels, labels)

	// Append provided Selector to already specified Selector in template
	service.Spec.Selector = util.MergeStringMapsOverwrite(service.Spec.Selector, selector)

	return service
}

// CreateConfigMapCHICommon creates new corev1.ConfigMap
func (c *Creator) CreateConfigMapCHICommon(options *ClickHouseConfigFilesGeneratorOptions) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CreateConfigMapCommonName(c.chi),
			Namespace: c.chi.Namespace,
			Labels:    c.labeler.getLabelsConfigMapCHICommon(),
		},
		// Data contains several sections which are to be several xml chopConfig files
		Data: c.chConfigFilesGenerator.CreateConfigFilesGroupCommon(options),
	}
}

// CreateConfigMapCHICommonUsers creates new corev1.ConfigMap
func (c *Creator) CreateConfigMapCHICommonUsers() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CreateConfigMapCommonUsersName(c.chi),
			Namespace: c.chi.Namespace,
			Labels:    c.labeler.getLabelsConfigMapCHICommonUsers(),
		},
		// Data contains several sections which are to be several xml chopConfig files
		Data: c.chConfigFilesGenerator.CreateConfigFilesGroupUsers(),
	}
}

// CreateConfigMapHost creates new corev1.ConfigMap
func (c *Creator) CreateConfigMapHost(host *chiv1.ChiHost) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CreateConfigMapPodName(host),
			Namespace: host.Address.Namespace,
			Labels:    c.labeler.getLabelsConfigMapHost(host),
		},
		Data: c.chConfigFilesGenerator.CreateConfigFilesGroupHost(host),
	}
}

// CreatePodDisruptionBudgetZooKeeper creates new policy.PodDisruptionBudget for zookeeper
func (c *Creator) CreatePodDisruptionBudgetZooKeeper(chi *chiv1.ClickHouseInstallation) *policyv1beta1.PodDisruptionBudget {
	zooKeeperPodDisruptionBudgetName := CreatePodDisruptionBudgetZooKeeperName(chi)

	// Create default PodDisruptionBudget
	maxUnavailable := intstr.IntOrString{
		Type:   intstr.Int,
		IntVal: (chi.Spec.Configuration.Zookeeper.Replica - 1) / 2,
	}
	return &policyv1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zooKeeperPodDisruptionBudgetName,
			Namespace: chi.Namespace,
			Labels:    c.labeler.getLabelsPodDisruptionBudgetZooKeeper(),
		},
		Spec: policyv1beta1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: c.labeler.getSelectorZooKeeperScope(),
			},
			MaxUnavailable: &maxUnavailable,
		},
	}
}

// CreateStatefulSet creates new apps.StatefulSet
func (c *Creator) CreateStatefulSet(host *chiv1.ChiHost) *apps.StatefulSet {
	statefulSetName := CreateStatefulSetName(host)
	serviceName := CreateStatefulSetServiceName(host)

	// Create apps.StatefulSet object
	replicasNum := host.GetStatefulSetReplicasNum()
	revisionHistoryLimit := int32(10)
	// StatefulSet has additional label - ZK config fingerprint
	statefulSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulSetName,
			Namespace: host.Address.Namespace,
			Labels:    c.labeler.getLabelsHostScope(host, true),
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    &replicasNum,
			ServiceName: serviceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: c.labeler.GetSelectorHostScope(host),
			},

			// IMPORTANT
			// Template is to be setup later
			Template: corev1.PodTemplateSpec{},

			// IMPORTANT
			// VolumeClaimTemplates are to be setup later
			VolumeClaimTemplates: nil,

			PodManagementPolicy: apps.OrderedReadyPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
			RevisionHistoryLimit: &revisionHistoryLimit,
		},
	}

	c.setupStatefulSetPodTemplate(statefulSet, host)
	c.setupStatefulSetVolumeClaimTemplates(statefulSet, host)
	c.setupStatefulSetVersion(statefulSet)

	host.StatefulSet = statefulSet

	return statefulSet
}

// setupStatefulSetVersion
// TODO property of the labeler?
func (c *Creator) setupStatefulSetVersion(statefulSet *apps.StatefulSet) {
	statefulSet.Labels = util.MergeStringMapsOverwrite(
		statefulSet.Labels,
		map[string]string{
			LabelObjectVersion: util.Fingerprint(statefulSet),
		},
	)
	c.a.V(2).F().Info("StatefulSet(%s/%s)\n%s", statefulSet.Namespace, statefulSet.Name, util.Dump(statefulSet))
}

// GetStatefulSetVersion
// TODO property of the labeler?
func (c *Creator) GetStatefulSetVersion(statefulSet *apps.StatefulSet) (string, bool) {
	if statefulSet == nil {
		return "", false
	}
	label, ok := statefulSet.Labels[LabelObjectVersion]
	return label, ok
}

// CreateStatefulSetZooKeeper creates new apps.StatefulSet
func (c *Creator) CreateStatefulSetZooKeeper(chi *chiv1.ClickHouseInstallation) *apps.StatefulSet {
	zooKeeperStatefulSetName := CreateStatefulSetZooKeeperName(chi)
	zooKeeperServiceName := CreateStatefulSetServiceZooKeeperServerName(chi)

	c.a.V(1).F().Info("Create StatefulSet %s/%s", chi.Namespace, zooKeeperServiceName)

	// Create apps.StatefulSet object
	replicasNum := chi.Spec.Configuration.Zookeeper.GetStatefulSetReplicasNum()
	revisionHistoryLimit := int32(10)

	// StatefulSet has additional label - ZK config fingerprint
	statefulSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zooKeeperStatefulSetName,
			Namespace: chi.Namespace,
			Labels:    c.labeler.getLabelsZooKeeperScope(),
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    &replicasNum,
			ServiceName: zooKeeperServiceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: c.labeler.getSelectorZooKeeperScope(),
			},

			// IMPORTANT
			// Template is to be setup later
			Template: corev1.PodTemplateSpec{},

			// IMPORTANT
			// VolumeClaimTemplates are to be setup later
			VolumeClaimTemplates: nil,

			PodManagementPolicy: apps.ParallelPodManagement,
			UpdateStrategy: apps.StatefulSetUpdateStrategy{
				Type: apps.RollingUpdateStatefulSetStrategyType,
			},
			RevisionHistoryLimit: &revisionHistoryLimit,
		},
	}

	c.setupStatefulSetZooKeeperPodTemplate(statefulSet, chi)
	c.setupStatefulSetZooKeeperVolumeClaimTemplates(statefulSet)

	return statefulSet
}

// PreparePersistentVolume
func (c *Creator) PreparePersistentVolume(pv *corev1.PersistentVolume, host *chiv1.ChiHost) *corev1.PersistentVolume {
	pv.Labels = util.MergeStringMapsOverwrite(pv.Labels, c.labeler.getLabelsHostScope(host, false))
	return pv
}

// setupStatefulSetPodTemplate performs PodTemplate setup of StatefulSet
func (c *Creator) setupStatefulSetPodTemplate(statefulSet *apps.StatefulSet, host *chiv1.ChiHost) {
	// Process Pod Template
	podTemplate := c.getPodTemplate(host)
	c.statefulSetApplyPodTemplate(statefulSet, podTemplate, host)

	// Post-process StatefulSet
	c.ensureStatefulSetTemplateIntegrity(statefulSet, host)
	c.personalizeStatefulSetTemplate(statefulSet, host)
}

// setupStatefulSetZooKeeperPodTemplate performs PodTemplate setup of StatefulSet
func (c *Creator) setupStatefulSetZooKeeperPodTemplate(statefulSet *apps.StatefulSet, chi *chiv1.ClickHouseInstallation) {
	podTemplate := c.getPodTemplateZooKeeper(chi)
	c.statefulSetApplyPodTemplateZooKeeper(statefulSet, podTemplate)
}

func (c *Creator) ensureStatefulSetTemplateIntegrity(statefulSet *apps.StatefulSet, host *chiv1.ChiHost) {
	c.ensureClickHouseContainerSpecified(statefulSet, host)
	c.ensureProbesSpecified(statefulSet)
	ensureNamedPortsSpecified(statefulSet, host)
}

func (c *Creator) ensureClickHouseContainerSpecified(statefulSet *apps.StatefulSet, _ *chiv1.ChiHost) {
	_, ok := getClickHouseContainer(statefulSet)
	if ok {
		return
	}

	// No ClickHouse container available, let's add one
	addContainer(
		&statefulSet.Spec.Template.Spec,
		c.newDefaultClickHouseContainer(),
	)
}

func (c *Creator) ensureProbesSpecified(statefulSet *apps.StatefulSet) {
	container, ok := getClickHouseContainer(statefulSet)
	if !ok {
		return
	}
	if container.LivenessProbe == nil {
		container.LivenessProbe = newDefaultLivenessProbe()
	}
	if container.ReadinessProbe == nil {
		container.ReadinessProbe = c.newDefaultReadinessProbe()
	}
}

func (c *Creator) personalizeStatefulSetTemplate(statefulSet *apps.StatefulSet, host *chiv1.ChiHost) {
	statefulSetName := CreateStatefulSetName(host)

	// Ensure pod created by this StatefulSet has alias 127.0.0.1
	statefulSet.Spec.Template.Spec.HostAliases = []corev1.HostAlias{
		{
			IP:        "127.0.0.1",
			Hostnames: []string{CreatePodHostname(host)},
		},
	}

	// Setup volumes based on ConfigMaps into Pod Template
	c.setupConfigMapVolumes(statefulSet, host)

	// In case we have default LogVolumeClaimTemplate specified - need to append log container to Pod Template
	if host.Templates.LogVolumeClaimTemplate != "" {
		addContainer(&statefulSet.Spec.Template.Spec, newDefaultLogContainer())
		c.a.V(1).F().Info("add log container for statefulSet %s", statefulSetName)
	}
}

// getPodTemplate gets Pod Template to be used to create StatefulSet
func (c *Creator) getPodTemplate(host *chiv1.ChiHost) *chiv1.ChiPodTemplate {
	statefulSetName := CreateStatefulSetName(host)

	// Which pod template would be used - either explicitly defined in or a default one
	podTemplate, ok := host.GetPodTemplate()
	if ok {
		// Host references known PodTemplate
		// Make local copy of this PodTemplate, in order not to spoil the original common-used template
		podTemplate = podTemplate.DeepCopy()
		c.a.V(1).F().Info("statefulSet %s use custom template %s", statefulSetName, podTemplate.Name)
	} else {
		// Host references UNKNOWN PodTemplate, will use default one
		podTemplate = c.newDefaultPodTemplate(statefulSetName)
		c.a.V(1).F().Info("statefulSet %s use default generated template", statefulSetName)
	}

	// Here we have local copy of Pod Template, to be used to create StatefulSet
	// Now we can customize this Pod Template for particular host

	c.labeler.prepareAffinity(podTemplate, host)

	return podTemplate
}

// getPodTemplateZooKeeper gets ZooKeeper Pod Template to be used to create StatefulSet
func (c *Creator) getPodTemplateZooKeeper(chi *chiv1.ClickHouseInstallation) *chiv1.ChiPodTemplate {
	statefulSetName := CreateStatefulSetZooKeeperName(chi)
	zooKeeperConfig := chi.Spec.Configuration.Zookeeper

	// TODO: use template if exists.
	// if template, ok := host.GetPodZooKeeperTemplate(); ok {
	//     return c.createPodZooKeeperFromTemplate()
	// }

	// Create default one
	podTemplate := c.newDefaultZooKeeperPodTemplate(zooKeeperConfig, statefulSetName)
	c.a.V(1).F().Info("statefulSet %s use default generated template", statefulSetName)

	return podTemplate
}

// setupConfigMapVolumes adds to each container in the Pod VolumeMount objects with
func (c *Creator) setupConfigMapVolumes(statefulSetObject *apps.StatefulSet, host *chiv1.ChiHost) {
	configMapPodName := CreateConfigMapPodName(host)
	configMapCommonName := CreateConfigMapCommonName(c.chi)
	configMapCommonUsersName := CreateConfigMapCommonUsersName(c.chi)

	// Add all ConfigMap objects as Volume objects of type ConfigMap
	statefulSetObject.Spec.Template.Spec.Volumes = append(
		statefulSetObject.Spec.Template.Spec.Volumes,
		newVolumeForConfigMap(configMapCommonName),
		newVolumeForConfigMap(configMapCommonUsersName),
		newVolumeForConfigMap(configMapPodName),
	)

	// And reference these Volumes in each Container via VolumeMount
	// So Pod will have ConfigMaps mounted as Volumes
	for i := range statefulSetObject.Spec.Template.Spec.Containers {
		// Convenience wrapper
		container := &statefulSetObject.Spec.Template.Spec.Containers[i]
		// Append to each Container current VolumeMount's to VolumeMount's declared in template
		container.VolumeMounts = append(
			container.VolumeMounts,
			newVolumeMount(configMapCommonName, dirPathCommonConfig),
			newVolumeMount(configMapCommonUsersName, dirPathUsersConfig),
			newVolumeMount(configMapPodName, dirPathHostConfig),
		)
	}
}

// setupStatefulSetApplyVolumeMounts applies `volumeMounts` of a `container`
func (c *Creator) setupStatefulSetApplyVolumeMounts(statefulSet *apps.StatefulSet, host *chiv1.ChiHost) {
	// Deal with `volumeMounts` of a `container`, located by the path:
	// .spec.templates.podTemplates.*.spec.containers.volumeMounts.*
	// VolumeClaimTemplates, that are directly referenced in Containers' VolumeMount object(s)
	// are appended to StatefulSet's Spec.VolumeClaimTemplates slice
	for i := range statefulSet.Spec.Template.Spec.Containers {
		// Convenience wrapper
		container := &statefulSet.Spec.Template.Spec.Containers[i]
		for j := range container.VolumeMounts {
			// Convenience wrapper
			volumeMount := &container.VolumeMounts[j]
			if volumeClaimTemplate, ok := c.chi.GetVolumeClaimTemplate(volumeMount.Name); ok {
				// Found VolumeClaimTemplate to mount by VolumeMount
				c.statefulSetAppendPVCTemplate(host, statefulSet, volumeClaimTemplate)
			}
		}
	}
}

// setupStatefulSetApplyVolumeClaimTemplates applies Data and Log VolumeClaimTemplates on all containers
func (c *Creator) setupStatefulSetApplyVolumeClaimTemplates(statefulSet *apps.StatefulSet, host *chiv1.ChiHost) {
	// Mount all named (data and log so far) VolumeClaimTemplates into all containers
	for i := range statefulSet.Spec.Template.Spec.Containers {
		// Convenience wrapper
		container := &statefulSet.Spec.Template.Spec.Containers[i]
		_ = c.setupStatefulSetApplyVolumeMount(host, statefulSet, container.Name, newVolumeMount(host.Templates.DataVolumeClaimTemplate, dirPathClickHouseData))
		_ = c.setupStatefulSetApplyVolumeMount(host, statefulSet, container.Name, newVolumeMount(host.Templates.LogVolumeClaimTemplate, dirPathClickHouseLog))
	}
}

// setupStatefulSetVolumeClaimTemplates performs VolumeClaimTemplate setup for Containers in PodTemplate of a StatefulSet
func (c *Creator) setupStatefulSetVolumeClaimTemplates(statefulSet *apps.StatefulSet, host *chiv1.ChiHost) {
	c.setupStatefulSetApplyVolumeMounts(statefulSet, host)
	c.setupStatefulSetApplyVolumeClaimTemplates(statefulSet, host)
}

// setupStatefulSetZooKeeperVolumeClaimTemplates performs VolumeClaimTemplate setup for Containers in PodTemplate of a StatefulSet
func (c *Creator) setupStatefulSetZooKeeperVolumeClaimTemplates(statefulSet *apps.StatefulSet) {
	// applies `volumeMounts` of a `container`
	// c.setupStatefulSetApplyVolumeMounts(statefulSet, host)
	for i := range statefulSet.Spec.Template.Spec.Containers {
		// Convenience wrapper
		container := &statefulSet.Spec.Template.Spec.Containers[i]

		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "data",
				MountPath: "/var/lib/zookeeper",
			},
		}
	}

	// applies Data VolumeClaimTemplates on all containers
	// c.setupStatefulSetApplyVolumeClaimTemplates(statefulSet, host)
	statefulSet.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "data",
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{
					corev1.ReadWriteOnce,
				},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: *resource.NewScaledQuantity(10, resource.Giga),
					},
				},
			},
		},
	}
}

// statefulSetApplyPodTemplate fills StatefulSet.Spec.Template with data from provided ChiPodTemplate
func (c *Creator) statefulSetApplyPodTemplate(
	statefulSet *apps.StatefulSet,
	template *chiv1.ChiPodTemplate,
	host *chiv1.ChiHost,
) {
	// StatefulSet's pod template is not directly compatible with ChiPodTemplate,
	// we need to extract some fields from ChiPodTemplate and apply on StatefulSet
	statefulSet.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name: template.Name,
			Labels: util.MergeStringMapsOverwrite(
				c.labeler.getLabelsHostScopeReady(host, true),
				template.ObjectMeta.Labels,
			),
			Annotations: util.MergeStringMapsOverwrite(
				c.labeler.getAnnotationsHostScope(host),
				template.ObjectMeta.Annotations,
			),
		},
		Spec: *template.Spec.DeepCopy(),
	}
}

// statefulSetApplyPodTemplateZooKeeper fills StatefulSet.Spec.Template with data from provided ChiPodTemplate
func (c *Creator) statefulSetApplyPodTemplateZooKeeper(
	statefulSet *apps.StatefulSet,
	template *chiv1.ChiPodTemplate,
) {
	// StatefulSet's pod template is not directly compatible with ChiPodTemplate,
	// we need to extract some fields from ChiPodTemplate and apply on StatefulSet
	statefulSet.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name: template.Name,
			Labels: util.MergeStringMapsOverwrite(
				c.labeler.getLabelsZooKeeperScope(),
				template.ObjectMeta.Labels,
			),
			Annotations: util.MergeStringMapsOverwrite(
				c.labeler.getAnnotationsZooKeeperScope(),
				template.ObjectMeta.Annotations,
			),
		},
		Spec: *template.Spec.DeepCopy(),
	}
}

func getClickHouseContainer(statefulSet *apps.StatefulSet) (*corev1.Container, bool) {
	// Find by name
	for i := range statefulSet.Spec.Template.Spec.Containers {
		container := &statefulSet.Spec.Template.Spec.Containers[i]
		if container.Name == ClickHouseContainerName {
			return container, true
		}
	}

	// Find by index
	if len(statefulSet.Spec.Template.Spec.Containers) > 0 {
		return &statefulSet.Spec.Template.Spec.Containers[0], true
	}

	return nil, false
}

func getClickHouseContainerStatus(pod *corev1.Pod) (*corev1.ContainerStatus, bool) {
	// Find by name
	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]
		if status.Name == ClickHouseContainerName {
			return status, true
		}
	}

	// Find by index
	if len(pod.Status.ContainerStatuses) > 0 {
		return &pod.Status.ContainerStatuses[0], true
	}

	return nil, false
}

// IsStatefulSetGeneration returns whether StatefulSet has requested generation or not
func IsStatefulSetGeneration(statefulSet *apps.StatefulSet, generation int64) bool {
	if statefulSet == nil {
		return false
	}

	// StatefulSet has .spec generation we are looking for
	return (statefulSet.Generation == generation) &&
		// and this .spec generation is being applied to replicas - it is observed right now
		(statefulSet.Status.ObservedGeneration == statefulSet.Generation) &&
		// and all replicas are of expected generation
		(statefulSet.Status.CurrentReplicas == *statefulSet.Spec.Replicas) &&
		// and all replicas are updated - meaning rolling update completed over all replicas
		(statefulSet.Status.UpdatedReplicas == *statefulSet.Spec.Replicas) &&
		// and current revision is an updated one - meaning rolling update completed over all replicas
		(statefulSet.Status.CurrentRevision == statefulSet.Status.UpdateRevision)
}

// IsStatefulSetReady returns whether StatefulSet is ready
func IsStatefulSetReady(statefulSet *apps.StatefulSet) bool {
	if statefulSet == nil {
		return false
	}

	if statefulSet.Spec.Replicas == nil {
		return false
	}
	// All replicas are in "Ready" status - meaning ready to be used - no failure inside
	return statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas
}

// IsStatefulSetNotReady returns whether StatefulSet is not ready
func IsStatefulSetNotReady(statefulSet *apps.StatefulSet) bool {
	if statefulSet == nil {
		return false
	}

	return !IsStatefulSetReady(statefulSet)
}

// StrStatefulSetStatus returns human-friendly string representation of StatefulSet status
func StrStatefulSetStatus(status *apps.StatefulSetStatus) string {
	return fmt.Sprintf(
		"ObservedGeneration:%d Replicas:%d ReadyReplicas:%d CurrentReplicas:%d UpdatedReplicas:%d CurrentRevision:%s UpdateRevision:%s",
		status.ObservedGeneration,
		status.Replicas,
		status.ReadyReplicas,
		status.CurrentReplicas,
		status.UpdatedReplicas,
		status.CurrentRevision,
		status.UpdateRevision,
	)
}

func ensureNamedPortsSpecified(statefulSet *apps.StatefulSet, host *chiv1.ChiHost) {
	// Ensure ClickHouse container has all named ports specified
	container, ok := getClickHouseContainer(statefulSet)
	if !ok {
		return
	}
	ensurePortByName(container, chDefaultTCPPortName, host.TCPPort)
	ensurePortByName(container, chDefaultHTTPPortName, host.HTTPPort)
	ensurePortByName(container, chDefaultInterserverHTTPPortName, host.InterserverHTTPPort)
}

func ensurePortByName(container *corev1.Container, name string, port int32) {
	// Find port with specified name
	for i := range container.Ports {
		containerPort := &container.Ports[i]
		if containerPort.Name == name {
			containerPort.HostPort = 0
			containerPort.ContainerPort = port
			return
		}
	}

	// Port with specified name not found. Need to append
	container.Ports = append(container.Ports, corev1.ContainerPort{
		Name:          name,
		ContainerPort: port,
	})
}

// setupStatefulSetApplyVolumeMount applies .templates.volumeClaimTemplates.* to a StatefulSet
func (c *Creator) setupStatefulSetApplyVolumeMount(
	host *chiv1.ChiHost,
	statefulSet *apps.StatefulSet,
	containerName string,
	volumeMount corev1.VolumeMount,
) error {

	// Sanity checks

	// 1. mountPath has to be reasonable
	if volumeMount.MountPath == "" {
		// No mount path specified
		return nil
	}

	volumeClaimTemplateName := volumeMount.Name

	// 2. volumeClaimTemplateName has to be reasonable
	if volumeClaimTemplateName == "" {
		// No VolumeClaimTemplate specified
		return nil
	}

	// 3. Specified (by volumeClaimTemplateName) VolumeClaimTemplate has to be available as well
	if _, ok := c.chi.GetVolumeClaimTemplate(volumeClaimTemplateName); !ok {
		// Incorrect/unknown .templates.VolumeClaimTemplate specified
		c.a.V(1).F().Warning("Can not find volumeClaimTemplate %s. Volume claim can not be mounted", volumeClaimTemplateName)
		return nil
	}

	// 4. Specified container has to be available
	container := getContainerByName(statefulSet, containerName)
	if container == nil {
		c.a.V(1).F().Warning("Can not find container %s. Volume claim can not be mounted", containerName)
		return nil
	}

	// Looks like all components are in place

	// Mount specified (by volumeMount.Name) VolumeClaimTemplate into volumeMount.Path (say into '/var/lib/clickhouse')
	//
	// A container wants to have this VolumeClaimTemplate mounted into `mountPath` in case:
	// 1. This VolumeClaimTemplate is NOT already mounted in the container with any VolumeMount (to avoid double-mount of a VolumeClaimTemplate)
	// 2. And specified `mountPath` (say '/var/lib/clickhouse') is NOT already mounted with any VolumeMount (to avoid double-mount/rewrite into single `mountPath`)

	for i := range container.VolumeMounts {
		// Convenience wrapper
		existingVolumeMount := &container.VolumeMounts[i]

		// 1. Check whether this VolumeClaimTemplate is already listed in VolumeMount of this container
		if volumeMount.Name == existingVolumeMount.Name {
			// This .templates.VolumeClaimTemplate is already used in VolumeMount
			c.a.V(1).F().Warning(
				"StatefulSet:%s container:%s volumeClaimTemplateName:%s already used",
				statefulSet.Name,
				container.Name,
				volumeMount.Name,
			)
			return nil
		}

		// 2. Check whether `mountPath` (say '/var/lib/clickhouse') is already mounted
		if volumeMount.MountPath == existingVolumeMount.MountPath {
			// `mountPath` (say /var/lib/clickhouse) is already mounted
			c.a.V(1).F().Warning(
				"StatefulSet:%s container:%s mountPath:%s already used",
				statefulSet.Name,
				container.Name,
				volumeMount.MountPath,
			)
			return nil
		}
	}

	// This VolumeClaimTemplate is not used explicitly by name and `mountPath` (say /var/lib/clickhouse) is not used also.
	// Let's mount this VolumeClaimTemplate into `mountPath` (say '/var/lib/clickhouse') of a container
	if template, ok := c.chi.GetVolumeClaimTemplate(volumeClaimTemplateName); ok {
		// Add VolumeClaimTemplate to StatefulSet
		c.statefulSetAppendPVCTemplate(host, statefulSet, template)
		// Add VolumeMount to ClickHouse container to `mountPath` point
		container.VolumeMounts = append(
			container.VolumeMounts,
			volumeMount,
		)
	}

	c.a.V(1).F().Info(
		"StatefulSet:%s container:%s mounted %s on %s",
		statefulSet.Name,
		container.Name,
		volumeMount.Name,
		volumeMount.MountPath,
	)

	return nil
}

// statefulSetAppendPVCTemplate appends to StatefulSet.Spec.VolumeClaimTemplates new entry with data from provided 'src' ChiVolumeClaimTemplate
func (c *Creator) statefulSetAppendPVCTemplate(
	host *chiv1.ChiHost,
	statefulSet *apps.StatefulSet,
	volumeClaimTemplate *chiv1.ChiVolumeClaimTemplate,
) {
	// Ensure VolumeClaimTemplates slice is in place
	if statefulSet.Spec.VolumeClaimTemplates == nil {
		statefulSet.Spec.VolumeClaimTemplates = make([]corev1.PersistentVolumeClaim, 0, 0)
	}

	// Check whether this VolumeClaimTemplate is already listed in statefulSet.Spec.VolumeClaimTemplates
	for i := range statefulSet.Spec.VolumeClaimTemplates {
		// Convenience wrapper
		volumeClaimTemplates := &statefulSet.Spec.VolumeClaimTemplates[i]
		if volumeClaimTemplates.Name == volumeClaimTemplate.Name {
			// This VolumeClaimTemplate is already listed in statefulSet.Spec.VolumeClaimTemplates
			// No need to add it second time
			return
		}
	}

	// VolumeClaimTemplate is not listed in statefulSet.Spec.VolumeClaimTemplates - let's add it
	persistentVolumeClaim := corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeClaimTemplate.Name,
			// TODO
			//  this has to wait until proper disk inheritance procedure will be available
			// UPDATE
			//  we are close to proper disk inheritance
			// Right now we hit the following error:
			// "Forbidden: updates to statefulset spec for fields other than 'replicas', 'template', and 'updateStrategy' are forbidden"
			Labels: c.labeler.getLabelsHostScope(host, false),
		},
		Spec: *volumeClaimTemplate.Spec.DeepCopy(),
	}
	// TODO introduce normalization
	volumeMode := corev1.PersistentVolumeFilesystem
	persistentVolumeClaim.Spec.VolumeMode = &volumeMode

	// Append copy of PersistentVolumeClaimSpec
	statefulSet.Spec.VolumeClaimTemplates = append(statefulSet.Spec.VolumeClaimTemplates, persistentVolumeClaim)
}

// newDefaultHostTemplate returns default Host Template to be used with StatefulSet
func newDefaultHostTemplate(name string) *chiv1.ChiHostTemplate {
	return &chiv1.ChiHostTemplate{
		Name: name,
		PortDistribution: []chiv1.ChiPortDistribution{
			{
				Type: chiv1.PortDistributionUnspecified,
			},
		},
		Spec: chiv1.ChiHost{
			Name:                "",
			TCPPort:             chPortNumberMustBeAssignedLater,
			HTTPPort:            chPortNumberMustBeAssignedLater,
			InterserverHTTPPort: chPortNumberMustBeAssignedLater,
			Templates:           chiv1.ChiTemplateNames{},
		},
	}
}

// newDefaultHostTemplateForHostNetwork
func newDefaultHostTemplateForHostNetwork(name string) *chiv1.ChiHostTemplate {
	return &chiv1.ChiHostTemplate{
		Name: name,
		PortDistribution: []chiv1.ChiPortDistribution{
			{
				Type: chiv1.PortDistributionClusterScopeIndex,
			},
		},
		Spec: chiv1.ChiHost{
			Name:                "",
			TCPPort:             chPortNumberMustBeAssignedLater,
			HTTPPort:            chPortNumberMustBeAssignedLater,
			InterserverHTTPPort: chPortNumberMustBeAssignedLater,
			Templates:           chiv1.ChiTemplateNames{},
		},
	}
}

// newDefaultPodTemplate returns default Pod Template to be used with StatefulSet
func (c *Creator) newDefaultPodTemplate(name string) *chiv1.ChiPodTemplate {
	podTemplate := &chiv1.ChiPodTemplate{
		Name: name,
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{},
			Volumes:    []corev1.Volume{},
		},
	}

	addContainer(
		&podTemplate.Spec,
		c.newDefaultClickHouseContainer(),
	)

	return podTemplate
}

// newDefaultZooKeeperPodTemplate returns default ZooKeeper Pod Template to be used with StatefulSet
func (c *Creator) newDefaultZooKeeperPodTemplate(zooKeeperConfig chiv1.ChiZookeeperConfig, name string) *chiv1.ChiPodTemplate {
	runAsUser := int64(1000)
	fSGroup := int64(1000)
	podTemplate := &chiv1.ChiPodTemplate{
		Name: name,
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				PodAntiAffinity: &corev1.PodAntiAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
						{
							Weight: 1,
							PodAffinityTerm: corev1.PodAffinityTerm{
								LabelSelector: &metav1.LabelSelector{
									MatchExpressions: []metav1.LabelSelectorRequirement{
										{
											Key:      "clickhouse.radondb.com/zookeeper",
											Operator: "In",
											Values: []string{
												name,
											},
										},
									},
								},
								TopologyKey: "kubernetes.io/hostname",
							},
						},
					},
				},
			},
			Containers: []corev1.Container{},
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: &runAsUser,
				FSGroup:   &fSGroup,
			},
		},
	}

	addContainer(
		&podTemplate.Spec,
		c.newDefaultZooKeeperContainer(zooKeeperConfig),
	)

	return podTemplate
}

// newDefaultLivenessProbe
func newDefaultLivenessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/ping",
				Port: intstr.Parse(chDefaultHTTPPortName), // What if it is not a default?
			},
		},
		InitialDelaySeconds: 60,
		PeriodSeconds:       3,
		FailureThreshold:    10,
	}
}

// newDefaultReadinessProbe
func (c *Creator) newDefaultReadinessProbe() *corev1.Probe {
	return nil
	//	return &corev1.Probe{
	//		Handler: corev1.Handler{
	//			HTTPGet: &corev1.HTTPGetAction{
	//				Path: "/replicas_status",
	//				Port: intstr.Parse(chDefaultHTTPPortName),
	//			},
	//		},
	//		InitialDelaySeconds: 10,
	//		PeriodSeconds:       3,
	//	}
}

// newDefaultClickHouseContainer returns default ClickHouse Container
func (c *Creator) newDefaultClickHouseContainer() corev1.Container {
	return corev1.Container{
		Name:  ClickHouseContainerName,
		Image: defaultClickHouseDockerImage,
		Ports: []corev1.ContainerPort{
			{
				Name:          chDefaultHTTPPortName,
				ContainerPort: chDefaultHTTPPortNumber,
			},
			{
				Name:          chDefaultTCPPortName,
				ContainerPort: chDefaultTCPPortNumber,
			},
			{
				Name:          chDefaultInterserverHTTPPortName,
				ContainerPort: chDefaultInterserverHTTPPortNumber,
			},
		},
		LivenessProbe:  newDefaultLivenessProbe(),
		ReadinessProbe: c.newDefaultReadinessProbe(),
	}
}

// newDefaultZooKeeperContainer returns default ZooKeeper Container
func (c *Creator) newDefaultZooKeeperContainer(zookeeperConfig chiv1.ChiZookeeperConfig) corev1.Container {
	b := &bytes.Buffer{}
	util.Iline(b, 0, "HOST=$(hostname -s) &&")
	util.Iline(b, 0, "DOMAIN=$(hostname -d) &&")
	util.Iline(b, 0, "ZOO_DATA_DIR=/var/lib/zookeeper/data &&")
	util.Iline(b, 0, "ZOO_DATA_LOG_DIR=/var/lib/zookeeper/datalog &&")
	util.Iline(b, 0, "SERVERS="+fmt.Sprintf("%d", zookeeperConfig.Replica)+" &&")
	util.Iline(b, 0, "CLIENT_PORT="+fmt.Sprintf("%d", zookeeperConfig.Port)+" &&")
	util.Iline(b, 0, "SERVER_PORT="+fmt.Sprintf("%d", zkDefaultServerPortNumber)+" &&")
	util.Iline(b, 0, "ELECTION_PORT="+fmt.Sprintf("%d", zkDefaultLeaderElectionPortNumber)+" &&")
	util.Iline(b, 0, "{")
	util.Iline(b, 0, "  echo clientPort=${CLIENT_PORT}")
	util.Iline(b, 0, "  echo tickTime=2000")
	util.Iline(b, 0, "  echo initLimit=300")
	util.Iline(b, 0, "  echo syncLimit=10")
	util.Iline(b, 0, "  echo maxClientCnxns=2000")
	util.Iline(b, 0, "  echo maxSessionTimeout=60000000")
	util.Iline(b, 0, "  echo dataDir=${ZOO_DATA_DIR}")
	util.Iline(b, 0, "  echo dataLogDir=${ZOO_DATA_LOG_DIR}")
	util.Iline(b, 0, "  echo autopurge.snapRetainCount=10")
	util.Iline(b, 0, "  echo autopurge.purgeInterval=1")
	util.Iline(b, 0, "  echo preAllocSize=131072")
	util.Iline(b, 0, "  echo snapCount=3000000")
	util.Iline(b, 0, "  echo leaderServes=yes")
	util.Iline(b, 0, "  echo standaloneEnabled="+strconv.FormatBool(zookeeperConfig.Replica == 1))
	util.Iline(b, 0, "  echo 4lw.commands.whitelist=*")
	util.Iline(b, 0, "} > /conf/zoo.cfg &&")
	util.Iline(b, 0, "{")
	util.Iline(b, 0, "  echo zookeeper.root.logger=CONSOLE")
	util.Iline(b, 0, "  echo zookeeper.console.threshold=INFO")
	util.Iline(b, 0, "  echo log4j.appender.CONSOLE=org.apache.log4j.ConsoleAppender")
	util.Iline(b, 0, "  echo log4j.appender.CONSOLE.layout=org.apache.log4j.PatternLayout")
	util.Iline(b, 0, "  echo 'log4j.rootLogger=${zookeeper.root.logger}'")
	util.Iline(b, 0, "  echo 'log4j.appender.CONSOLE.Threshold=${zookeeper.console.threshold}'")
	util.Iline(b, 0, "  echo 'log4j.appender.CONSOLE.layout.ConversionPattern=%%d{ISO8601} [myid:%%X{myid}] - %%-5p [%%t:%%C{1}@%%L] - %%m%%n'")
	util.Iline(b, 0, "} > /conf/log4j.properties &&")
	util.Iline(b, 0, "echo JVMFLAGS='-Xms128M -Xmx4G -XX:+CMSParallelRemarkEnabled' > /conf/java.env &&")
	util.Iline(b, 0, "if [[ $HOST =~ (.*)-([0-9]+)$ ]]; then")
	util.Iline(b, 0, "    NAME=${BASH_REMATCH[1]}")
	util.Iline(b, 0, "    ORD=${BASH_REMATCH[2]}")
	util.Iline(b, 0, "else")
	util.Iline(b, 0, "    echo 'Fialed to parse name and ordinal of Pod'")
	util.Iline(b, 0, "    exit 1")
	util.Iline(b, 0, "fi &&")
	util.Iline(b, 0, "mkdir -p ${ZOO_DATA_DIR} &&")
	util.Iline(b, 0, "mkdir -p ${ZOO_DATA_LOG_DIR} &&")
	util.Iline(b, 0, "export MY_ID=$((ORD+1)) &&")
	util.Iline(b, 0, "echo $MY_ID > $ZOO_DATA_DIR/myid &&")
	util.Iline(b, 0, "for (( i=1; i<=$SERVERS; i++ )); do")
	util.Iline(b, 0, "  echo server.$i=$NAME-$((i-1)).$DOMAIN:$SERVER_PORT:$ELECTION_PORT >> /conf/zoo.cfg")
	util.Iline(b, 0, "done &&")
	util.Iline(b, 0, "chown -Rv zookeeper \"$ZOO_DATA_DIR\" \"$ZOO_DATA_LOG_DIR\" \"$ZOO_LOG_DIR\" \"$ZOO_CONF_DIR\" &&")
	util.Iline(b, 0, "cat /conf/zoo.cfg &&")
	util.Iline(b, 0, "zkServer.sh start-foreground")

	return corev1.Container{
		Name:            zooKeeperContainerName,
		Image:           defaultZooKeeperDockerImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{
			{
				Name:          zkDefaultClientPortName,
				ContainerPort: zookeeperConfig.Port,
			},
			{
				Name:          zkDefaultServerPortName,
				ContainerPort: zkDefaultServerPortNumber,
			},
			{
				Name:          zkDefaultLeaderElectionPortName,
				ContainerPort: zkDefaultLeaderElectionPortNumber,
			},
			{
				Name:          defaultPrometheusPortName,
				ContainerPort: defaultPrometheusPortNumber,
			},
		},
		Command: []string{
			"bash",
			"-x",
			"-c",
			b.String(),
		},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{Command: []string{
					"bash",
					"-c",
					"OK=$(echo ruok | nc 127.0.0.1 " +
						strconv.Itoa(int(zookeeperConfig.Port)) +
						"); if [[ \"$OK\" == \"imok\" ]]; then exit 0; else exit 1; fi",
				}},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      5,
		},
		LivenessProbe: &corev1.Probe{
			Handler: corev1.Handler{
				Exec: &corev1.ExecAction{Command: []string{
					"bash",
					"-c",
					"OK=$(echo ruok | nc 127.0.0.1 " +
						strconv.Itoa(int(zookeeperConfig.Port)) +
						"); if [[ \"$OK\" == \"imok\" ]]; then exit 0; else exit 1; fi",
				}},
			},
			InitialDelaySeconds: 10,
			TimeoutSeconds:      5,
		},
	}
}

// newDefaultLogContainer returns default Log Container
func newDefaultLogContainer() corev1.Container {
	return corev1.Container{
		Name:  ClickHouseLogContainerName,
		Image: defaultBusyBoxDockerImage,
		Command: []string{
			"/bin/sh", "-c", "--",
		},
		Args: []string{
			"while true; do sleep 30; done;",
		},
	}
}

// addContainer adds container to ChiPodTemplate
func addContainer(podSpec *corev1.PodSpec, container corev1.Container) {
	podSpec.Containers = append(podSpec.Containers, container)
}

// newVolumeForConfigMap returns corev1.Volume object with defined name
func newVolumeForConfigMap(name string) corev1.Volume {
	var defaultMode int32 = 0644
	return corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: name,
				},
				DefaultMode: &defaultMode,
			},
		},
	}
}

// newVolumeMount returns corev1.VolumeMount object with name and mount path
func newVolumeMount(name, mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
	}
}

// getContainerByName finds Container with specified name among all containers of Pod Template in StatefulSet
func getContainerByName(statefulSet *apps.StatefulSet, name string) *corev1.Container {
	for i := range statefulSet.Spec.Template.Spec.Containers {
		// Convenience wrapper
		container := &statefulSet.Spec.Template.Spec.Containers[i]
		if container.Name == name {
			return container
		}
	}

	return nil
}
