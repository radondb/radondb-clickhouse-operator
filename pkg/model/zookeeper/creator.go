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
	"bytes"
	"fmt"
	"strconv"

	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	log "github.com/radondb/clickhouse-operator/pkg/announcer"
	chiv1 "github.com/radondb/clickhouse-operator/pkg/apis/clickhouse.radondb.com/v1"
	"github.com/radondb/clickhouse-operator/pkg/model"
	"github.com/radondb/clickhouse-operator/pkg/util"
)

// Creator specifies creator object
type Creator struct {
	chi     *chiv1.ClickHouseInstallation
	labeler *model.Labeler
	a       log.Announcer
}

// NewCreator creates new Creator object
func NewCreator(chi *chiv1.ClickHouseInstallation) *Creator {
	return &Creator{
		chi:     chi,
		labeler: model.NewLabeler(chi),
		a:       log.M(chi),
	}
}

// CreateServiceZooKeeper creates new corev1.Service for zookeeper
func (c *Creator) CreateServiceZooKeeper() *corev1.Service {
	serviceName := model.CreateStatefulSetServiceZooKeeperServerName(c.chi)
	ownerReferences := getOwnerReferences(c.chi.TypeMeta, c.chi.ObjectMeta, true, true)

	c.a.V(1).F().Info("%s/%s", c.chi.Namespace, serviceName)
	if template, ok := c.chi.GetZooKeeperServiceTemplate(); ok {
		return c.createServiceFromTemplate(
			template,
			c.chi.Namespace,
			serviceName,
			c.labeler.GetLabelsServiceZooKeeper(),
			c.labeler.GetSelectorZooKeeperScope(),
			ownerReferences,
		)
	}

	// Create default Service
	// We do not have .templates.ZooKeeperServiceTemplate specified or it is incorrect
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            serviceName,
			Namespace:       c.chi.Namespace,
			Labels:          c.labeler.GetLabelsServiceZooKeeper(),
			OwnerReferences: ownerReferences,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       zkDefaultServerPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       c.getPortNumberByName(zkDefaultServerPortName),
					TargetPort: intstr.FromString(zkDefaultServerPortName),
				},
				{
					Name:       zkDefaultLeaderElectionPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       c.getPortNumberByName(zkDefaultLeaderElectionPortName),
					TargetPort: intstr.FromString(zkDefaultLeaderElectionPortName),
				},
				{
					Name:       zkDefaultClientPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       c.getPortNumberByName(zkDefaultClientPortName),
					TargetPort: intstr.FromString(zkDefaultClientPortName),
				},
				{
					Name:       defaultPrometheusPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       c.getPortNumberByName(defaultPrometheusPortName),
					TargetPort: intstr.FromString(defaultPrometheusPortName),
				},
			},
			Selector: c.labeler.GetSelectorZooKeeperScope(),
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
	model.MakeObjectVersionLabel(&svc.ObjectMeta, svc)
	return svc
}

// CreateStatefulSetZooKeeper creates new apps.StatefulSet
func (c *Creator) CreateStatefulSetZooKeeper() *apps.StatefulSet {
	statefulSetName := model.CreateStatefulSetZooKeeperName(c.chi)
	serviceName := model.CreateStatefulSetServiceZooKeeperServerName(c.chi)
	ownerReferences := getOwnerReferences(c.chi.TypeMeta, c.chi.ObjectMeta, true, true)

	// Create apps.StatefulSet object
	revisionHistoryLimit := int32(10)
	replicasNum := c.chi.Spec.Configuration.Zookeeper.Replica

	// StatefulSet has additional label - ZK config fingerprint
	statefulSet := &apps.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            statefulSetName,
			Namespace:       c.chi.Namespace,
			Labels:          c.labeler.GetLabelsZooKeeperScope(),
			OwnerReferences: ownerReferences,
		},
		Spec: apps.StatefulSetSpec{
			Replicas:    &replicasNum,
			ServiceName: serviceName,
			Selector: &metav1.LabelSelector{
				MatchLabels: c.labeler.GetSelectorZooKeeperScope(),
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

	c.setupStatefulSetZooKeeperPodTemplate(statefulSet)
	c.setupStatefulSetZooKeeperVolumeClaimTemplates(statefulSet)
	c.setupStatefulSetVersion(statefulSet)

	return statefulSet
}

// setupStatefulSetZooKeeperPodTemplate performs PodTemplate setup of StatefulSet
func (c *Creator) setupStatefulSetZooKeeperPodTemplate(statefulSet *apps.StatefulSet) {
	podTemplate := c.getPodTemplateZooKeeper()
	c.statefulSetApplyPodTemplateZooKeeper(statefulSet, podTemplate)

	// Post-process StatefulSet
	c.ensureStatefulSetZooKeeperTemplateIntegrity(statefulSet)
}

// getPodTemplateZooKeeper gets ZooKeeper Pod Template to be used to create StatefulSet
func (c *Creator) getPodTemplateZooKeeper() *chiv1.ChiPodTemplate {
	statefulSetName := model.CreateStatefulSetZooKeeperName(c.chi)

	// Which pod template would be used - either explicitly defined in or a default one
	podTemplate, ok := c.chi.GetZooKeeperPodTemplate()
	if ok {
		// Host references known PodTemplate
		// Make local copy of this PodTemplate, in order not to spoil the original common-used template
		podTemplate = podTemplate.DeepCopy()
		c.a.V(1).F().Info("statefulSet %s use custom template %s", statefulSetName, podTemplate.Name)
	} else {
		// Host references UNKNOWN PodTemplate, will use default one
		podTemplate = c.newDefaultZooKeeperPodTemplate(statefulSetName)
		c.a.V(1).F().Info("statefulSet %s use default generated template", statefulSetName)
	}

	// Here we have local copy of Pod Template, to be used to create StatefulSet
	// Now we can customize this Pod Template
	// prepareAffinity(podTemplate, host)

	return podTemplate
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
				c.labeler.GetLabelsZooKeeperScope(),
				template.ObjectMeta.Labels,
			),
			Annotations: template.ObjectMeta.Annotations,
		},
		Spec: *template.Spec.DeepCopy(),
	}

	if statefulSet.Spec.Template.Spec.TerminationGracePeriodSeconds == nil {
		terminationGracePeriod := int64(60)
		statefulSet.Spec.Template.Spec.TerminationGracePeriodSeconds = &terminationGracePeriod
	}
}

// ensureStatefulSetZooKeeperTemplateIntegrity
func (c *Creator) ensureStatefulSetZooKeeperTemplateIntegrity(statefulSet *apps.StatefulSet) {
	c.ensureZooKeeperContainerSpecified(statefulSet)
	c.ensureZooKeeperProbesSpecified(statefulSet)
	c.ensureZooKeeperNamedPortsSpecified(statefulSet)
	c.ensureZooKeeperCommandSpecified(statefulSet)
}

// ensureZooKeeperContainerSpecified
func (c *Creator) ensureZooKeeperContainerSpecified(statefulSet *apps.StatefulSet) {
	_, ok := c.getZooKeeperContainer(statefulSet)
	if ok {
		return
	}

	// No ZooKeeper container available, let's add one
	addContainer(
		&statefulSet.Spec.Template.Spec,
		c.newDefaultZooKeeperContainer(),
	)
}

// ensureZooKeeperProbesSpecified
func (c *Creator) ensureZooKeeperProbesSpecified(statefulSet *apps.StatefulSet) {
	container, ok := c.getZooKeeperContainer(statefulSet)
	if !ok {
		return
	}
	if container.LivenessProbe == nil {
		container.LivenessProbe = c.newDefaultZooKeeperLivenessProbe()
	}
	if container.ReadinessProbe == nil {
		container.ReadinessProbe = c.newDefaultZooKeeperReadinessProbe()
	}
}

// ensureZooKeeperNamedPortsSpecified
func (c *Creator) ensureZooKeeperNamedPortsSpecified(statefulSet *apps.StatefulSet) {
	// Ensure ClickHouse container has all named ports specified
	container, ok := c.getZooKeeperContainer(statefulSet)
	if !ok {
		return
	}
	ensurePortByName(container, zkDefaultClientPortName, c.getPortNumberByName(zkDefaultClientPortName))
	ensurePortByName(container, zkDefaultServerPortName, c.getPortNumberByName(zkDefaultServerPortName))
	ensurePortByName(container, zkDefaultLeaderElectionPortName, c.getPortNumberByName(zkDefaultLeaderElectionPortName))
	ensurePortByName(container, defaultPrometheusPortName, c.getPortNumberByName(defaultPrometheusPortName))
}

// ensureZooKeeperCommandSpecified
func (c *Creator) ensureZooKeeperCommandSpecified(statefulSet *apps.StatefulSet) {
	// Ensure ClickHouse container has all named ports specified
	container, ok := c.getZooKeeperContainer(statefulSet)
	if !ok {
		return
	}

	container.Command = c.newDefaultZooKeeperCommand()
	// copy(container.Command, c.newDefaultZooKeeperCommand(statefulSet))
}

// setupStatefulSetZooKeeperVolumeClaimTemplates performs VolumeClaimTemplate setup for Containers in PodTemplate of a StatefulSet
func (c *Creator) setupStatefulSetZooKeeperVolumeClaimTemplates(statefulSet *apps.StatefulSet) {
	// applies `volumeMounts` of a `container`
	// c.setupStatefulSetApplyVolumeMounts(statefulSet, host)
	for i := range statefulSet.Spec.Template.Spec.Containers {
		// Convenience wrapper
		container := &statefulSet.Spec.Template.Spec.Containers[i]
		for j := range container.VolumeMounts {
			// Convenience wrapper
			volumeMount := &container.VolumeMounts[j]
			if volumeClaimTemplate, ok := c.chi.GetVolumeClaimTemplate(volumeMount.Name); ok {
				// Found VolumeClaimTemplate to mount by VolumeMount
				c.statefulSetZooKeeperAppendPVCTemplate(statefulSet, volumeClaimTemplate)
			}
		}

		// Mount all named (data and log so far) VolumeClaimTemplates into all containers
		for i := range statefulSet.Spec.Template.Spec.Containers {
			// Convenience wrapper
			container := &statefulSet.Spec.Template.Spec.Containers[i]
			_ = c.setupStatefulSetZooKeeperApplyVolumeMount(statefulSet, container.Name, newVolumeMount(c.chi.Spec.Defaults.Templates.GetZooKeeperVolumeClaimTemplate(), dirPathZooKeeperData))
		}
	}
}

// statefulSetZooKeeperAppendPVCTemplate appends to StatefulSet.Spec.VolumeClaimTemplates new entry with data from provided 'src' ChiVolumeClaimTemplate
func (c *Creator) statefulSetZooKeeperAppendPVCTemplate(
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
			Name:   volumeClaimTemplate.Name,
			Labels: c.labeler.GetLabelsZooKeeperScope(),
		},
		Spec: *volumeClaimTemplate.Spec.DeepCopy(),
	}
	// TODO introduce normalization
	volumeMode := corev1.PersistentVolumeFilesystem
	persistentVolumeClaim.Spec.VolumeMode = &volumeMode

	// Append copy of PersistentVolumeClaimSpec
	statefulSet.Spec.VolumeClaimTemplates = append(statefulSet.Spec.VolumeClaimTemplates, persistentVolumeClaim)
}

// setupStatefulSetZooKeeperApplyVolumeMount applies .templates.volumeClaimTemplates.* to a StatefulSet
func (c *Creator) setupStatefulSetZooKeeperApplyVolumeMount(
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
		c.statefulSetZooKeeperAppendPVCTemplate(statefulSet, template)
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

// NewPodDisruptionBudgetZooKeeper creates new policy.PodDisruptionBudget for zookeeper
func (c *Creator) NewPodDisruptionBudgetZooKeeper(chi *chiv1.ClickHouseInstallation) *v1beta1.PodDisruptionBudget {
	zooKeeperPodDisruptionBudgetName := model.CreatePodDisruptionBudgetZooKeeperName(chi)

	return &v1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      zooKeeperPodDisruptionBudgetName,
			Namespace: chi.Namespace,
			Labels:    c.labeler.GetLabelsPodDisruptionBudgetZooKeeper(),
		},
		Spec: v1beta1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: c.labeler.GetSelectorZooKeeperScope(),
			},
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: (chi.Spec.Configuration.Zookeeper.Replica - 1) / 2,
			},
		},
	}
}

// createServiceFromTemplate create Service from ChiServiceTemplate and additional info
func (c *Creator) createServiceFromTemplate(
	template *chiv1.ChiServiceTemplate,
	namespace string,
	name string,
	labels map[string]string,
	selector map[string]string,
	ownerReferences []metav1.OwnerReference,
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
	service.OwnerReferences = ownerReferences

	// Append provided Labels to already specified Labels in template
	service.Labels = util.MergeStringMapsOverwrite(service.Labels, labels)

	// Append provided Selector to already specified Selector in template
	service.Spec.Selector = util.MergeStringMapsOverwrite(service.Spec.Selector, selector)

	// And after the object is ready we can put version label
	model.MakeObjectVersionLabel(&service.ObjectMeta, service)

	return service
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

// setupStatefulSetVersion
// TODO property of the labeler?
func (c *Creator) setupStatefulSetVersion(statefulSet *apps.StatefulSet) {
	statefulSet.Labels = util.MergeStringMapsOverwrite(
		statefulSet.Labels,
		map[string]string{
			model.LabelObjectVersion: util.Fingerprint(statefulSet),
		},
	)
	c.a.V(2).F().Info("StatefulSet(%s/%s)\n%s", statefulSet.Namespace, statefulSet.Name, util.Dump(statefulSet))
}

// GetStatefulSetVersion gets version of the StatefulSet
// TODO property of the labeler?
func (c *Creator) GetStatefulSetVersion(statefulSet *apps.StatefulSet) (string, bool) {
	if statefulSet == nil {
		return "", false
	}
	label, ok := statefulSet.Labels[model.LabelObjectVersion]
	return label, ok
}

// getZooKeeperContainer
func (c *Creator) getZooKeeperContainer(statefulSet *apps.StatefulSet) (*corev1.Container, bool) {
	// Find by name
	for i := range statefulSet.Spec.Template.Spec.Containers {
		container := &statefulSet.Spec.Template.Spec.Containers[i]
		if container.Name == zooKeeperContainerName {
			return container, true
		}
	}

	// Find by index
	if len(statefulSet.Spec.Template.Spec.Containers) > 0 {
		return &statefulSet.Spec.Template.Spec.Containers[0], true
	}

	return nil, false
}

// newDefaultZooKeeperPodTemplate returns default ZooKeeper Pod Template to be used with StatefulSet
func (c *Creator) newDefaultZooKeeperPodTemplate(name string) *chiv1.ChiPodTemplate {
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
		c.newDefaultZooKeeperContainer(),
	)

	return podTemplate
}

// newDefaultZooKeeperLivenessProbe
func (c *Creator) newDefaultZooKeeperLivenessProbe() *corev1.Probe {

	return &corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{Command: []string{
				"bash",
				"-c",
				"OK=$(echo ruok | nc 127.0.0.1 " +
					strconv.Itoa(int(c.getPortNumberByName(zkDefaultClientPortName))) +
					"); if [[ \"$OK\" == \"imok\" ]]; then exit 0; else exit 1; fi",
			}},
		},
		InitialDelaySeconds: 60,
		PeriodSeconds:       3,
		FailureThreshold:    10,
	}
}

// newDefaultZooKeeperReadinessProbe
func (c *Creator) newDefaultZooKeeperReadinessProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			Exec: &corev1.ExecAction{Command: []string{
				"bash",
				"-c",
				"OK=$(echo ruok | nc 127.0.0.1 " +
					strconv.Itoa(int(c.getPortNumberByName(zkDefaultClientPortName))) +
					"); if [[ \"$OK\" == \"imok\" ]]; then exit 0; else exit 1; fi",
			}},
		},
		InitialDelaySeconds: 10,
		PeriodSeconds:       3,
	}
}

// newDefaultZooKeeperCommand returns default ZooKeeper Command
func (c *Creator) newDefaultZooKeeperCommand() []string {
	b := &bytes.Buffer{}
	util.Iline(b, 0, "HOST=$(hostname -s) &&")
	util.Iline(b, 0, "DOMAIN=$(hostname -d) &&")
	util.Iline(b, 0, "ZOO_DATA_DIR=/var/lib/zookeeper/data &&")
	util.Iline(b, 0, "ZOO_DATA_LOG_DIR=/var/lib/zookeeper/datalog &&")
	util.Iline(b, 0, "SERVERS="+fmt.Sprintf("%d", c.chi.Spec.Configuration.Zookeeper.Replica)+" &&")
	util.Iline(b, 0, "CLIENT_PORT="+fmt.Sprintf("%d", c.getPortNumberByName(zkDefaultClientPortName))+" &&")
	util.Iline(b, 0, "SERVER_PORT="+fmt.Sprintf("%d", c.getPortNumberByName(zkDefaultServerPortName))+" &&")
	util.Iline(b, 0, "ELECTION_PORT="+fmt.Sprintf("%d", c.getPortNumberByName(zkDefaultLeaderElectionPortName))+" &&")
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
	util.Iline(b, 0, "  echo standaloneEnabled=false")
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

	return []string{
		"bash",
		"-x",
		"-c",
		b.String(),
	}
}

// newDefaultZooKeeperContainer returns default ZooKeeper Container
func (c *Creator) newDefaultZooKeeperContainer() corev1.Container {
	container := corev1.Container{
		Name:  zooKeeperContainerName,
		Image: defaultZooKeeperDockerImage,
		Ports: []corev1.ContainerPort{
			{
				Name:          zkDefaultClientPortName,
				ContainerPort: c.getPortNumberByName(zkDefaultClientPortName),
			},
			{
				Name:          zkDefaultServerPortName,
				ContainerPort: c.getPortNumberByName(zkDefaultServerPortName),
			},
			{
				Name:          zkDefaultLeaderElectionPortName,
				ContainerPort: c.getPortNumberByName(zkDefaultLeaderElectionPortName),
			},
			{
				Name:          defaultPrometheusPortName,
				ContainerPort: c.getPortNumberByName(defaultPrometheusPortName),
			},
		},
	}

	return container
}

// CreateStatefulSetZooKeeper creates new apps.StatefulSet
func (c *Creator) getPortNumberByName(name string) int32 {
	if template, ok := c.chi.GetZooKeeperServiceTemplate(); ok {
		for _, port := range template.Spec.Ports {
			if port.Name == name {
				return port.Port
			}
		}
	}

	if name == zkDefaultClientPortName {
		return zkDefaultClientPortNumber
	}
	if name == zkDefaultServerPortName {
		return zkDefaultServerPortNumber
	}
	if name == zkDefaultLeaderElectionPortName {
		return zkDefaultLeaderElectionPortNumber
	}
	if name == defaultPrometheusPortName {
		return defaultPrometheusPortNumber
	}

	return -1
}

// ensurePortByName
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

// addContainer adds container to ChiPodTemplate
func addContainer(podSpec *corev1.PodSpec, container corev1.Container) {
	podSpec.Containers = append(podSpec.Containers, container)
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

func getOwnerReferences(t metav1.TypeMeta, o metav1.ObjectMeta, controller, blockOwnerDeletion bool) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		{
			APIVersion:         t.APIVersion,
			Kind:               t.Kind,
			Name:               o.Name,
			UID:                o.UID,
			Controller:         &controller,
			BlockOwnerDeletion: &blockOwnerDeletion,
		},
	}
}
