package webhook

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	AnnotationInjected = "jayce.jia.latency.agent.injected"

	LabelComponent = "app.kubernetes.io/component"
)

type (
	Template  *corev1.Pod
	Templates map[string]string
)

// Get Sidecar Container Object
func NewSidecarContainer(agentConfig *AgentConfig) *corev1.Container {
	portInt := int(agentConfig.ManagenemtPort)
	privileged := true
	container := corev1.Container{
		Name:    agentConfig.ContainerName,
		Image:   fmt.Sprintf("%s:%s", agentConfig.Image, agentConfig.ImageTag),
		Command: []string{"./agent"},
		Args:    []string{"-port", strconv.Itoa(portInt), "-latency", agentConfig.InitLatency.String(), "-period", agentConfig.ApplyPeriod.String()},
		Ports: []corev1.ContainerPort{{
			Name:          "management",
			ContainerPort: agentConfig.ManagenemtPort,
			Protocol:      "TCP",
		}},
		ReadinessProbe: &corev1.Probe{
			Handler: corev1.Handler{HTTPGet: &corev1.HTTPGetAction{
				Path: "health",
				Port: intstr.FromInt(portInt),
			}},
			InitialDelaySeconds: 2,
			TimeoutSeconds:      2,
			PeriodSeconds:       1,
		},
		ImagePullPolicy: "IfNotPresent",
		SecurityContext: &corev1.SecurityContext{Privileged: &privileged},
	}
	return &container
}

// Do inject the sidecar to the pod
// Returns the raw string template, as well as the parse pod form
func InjectSidecar(params InjectionParameters) (mergedPod *corev1.Pod, err error) {
	metadata := &params.pod.ObjectMeta
	mergedPod = params.pod.DeepCopy()
	if mergedPod.Annotations == nil {
		mergedPod.Annotations = make(map[string]string)
	}
	if injected, f := metadata.Annotations[AnnotationInjected]; !f && injected == "true" {
		// already injected, return current pod spec
		return mergedPod, nil
	}

	containers := mergedPod.Spec.Containers
	for _, container := range containers {
		if container.Name == params.container.Name {
			return nil, fmt.Errorf("unable to inject sidecar, duplicated name found: %s", params.container.Name)
		}
	}
	// append the sidecar into the pod
	containers = append(containers, *params.container)
	mergedPod.Spec.Containers = containers
	// tag injected status
	mergedPod.Annotations[AnnotationInjected] = "true"

	return mergedPod, nil
}
