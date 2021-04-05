package webhook

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"gomodules.xyz/jsonpatch/v3"
	kubeApiAdmissionV1 "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func init() {
	_ = coreV1.AddToScheme(runtimeScheme)
	_ = kubeApiAdmissionV1.AddToScheme(runtimeScheme)
}

// Webhook implements a mutating webhook for automatic sidecar injection.
type Webhook struct {
	// sidecar config
	Config *AgentConfig
}

// AgentConfig configures parameters for the sidecar injection webhook.
type AgentConfig struct {
	// sidecar container name
	ContainerName string
	// sidecar image
	Image string
	// sidecar image tag
	ImageTag string
	// Agenet Management Port
	ManagenemtPort int32
	// Agent Initial Latency
	InitLatency time.Duration
	// Agent Period for Applying Configuration
	ApplyPeriod time.Duration
}

// NewWebhook creates a new instance of a mutating webhook for automatic sidecar injection.
func NewWebhook(config AgentConfig) *Webhook {
	return &Webhook{
		Config: &config,
	}
}

func (wh *Webhook) ServeInject(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		} else {
			klog.Error(err)
		}
	}
	klog.Info("Request Received.", string(body))
	if len(body) == 0 {
		http.Error(w, "no body found", http.StatusBadRequest)
		return
	}

	// The AdmissionReview that was sent to the webhook
	requestedAdmissionReview := v1beta1.AdmissionReview{}
	// The AdmissionReview that will be returned
	responseAdmissionReview := v1beta1.AdmissionReview{}

	if _, _, err := deserializer.Decode(body, nil, &requestedAdmissionReview); err != nil {
		responseAdmissionReview.Response = toAdmissionErrResponse(err)
	} else {
		responseAdmissionReview.Response = wh.inject(&requestedAdmissionReview)
	}

	responseAdmissionReview.APIVersion = requestedAdmissionReview.APIVersion
	responseAdmissionReview.TypeMeta = requestedAdmissionReview.TypeMeta
	if responseAdmissionReview.Response != nil {
		if requestedAdmissionReview.Request != nil {
			responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		}
	}

	respBytes, err := json.Marshal(responseAdmissionReview)
	if err != nil {
		klog.Error(err)
	}
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	}
}

func (wh *Webhook) inject(ar *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	req := ar.Request
	var pod coreV1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return toAdmissionErrResponse(err)
	}
	if !needsInjection(&pod) {
		// no need to do the injection
		return &v1beta1.AdmissionResponse{Allowed: true}
	}
	container := NewSidecarContainer(wh.Config)
	klog.Info("Container to inject:", container)

	if pod.ObjectMeta.Namespace == "" {
		pod.ObjectMeta.Namespace = req.Namespace
	}

	params := InjectionParameters{
		pod:       &pod,
		container: container,
	}

	patchBytes, err := injectPod(params)
	if err != nil {
		return toAdmissionErrResponse(err)
	}

	patchType := v1beta1.PatchTypeJSONPatch
	reviewResponse := v1beta1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}
	return &reviewResponse
}

func needsInjection(pod *coreV1.Pod) bool {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	if pod.Labels[LabelComponent] != "tidb" {
		// only inject into tidb pods
		return false
	}
	if injected, f := pod.Annotations[AnnotationInjected]; !f && injected == "true" {
		// already injected, return current pod spec
		return false
	}
	return true
}

type InjectionParameters struct {
	pod       *coreV1.Pod
	container *coreV1.Container
}

// injectPod is the core of the injection logic. This takes a pod and injection
// template, as well as some inputs to the injection template, and produces a
// JSON patch.
func injectPod(req InjectionParameters) ([]byte, error) {
	// Run the injection template, merge container into
	mergedPod, err := InjectSidecar(req)
	if err != nil {
		return nil, fmt.Errorf("failed to run injection template: %v", err)
	}

	patch, err := createPatch(mergedPod, req.pod)
	if err != nil {
		return nil, fmt.Errorf("failed to create patch: %v", err)
	}

	return patch, nil
}

func createPatch(pod, original *coreV1.Pod) ([]byte, error) {
	newPodJSON, err := json.Marshal(pod)
	if err != nil {
		return nil, err
	}
	originalPodJSON, err := json.Marshal(original)
	if err != nil {
		return nil, err
	}
	p, err := jsonpatch.CreatePatch(originalPodJSON, newPodJSON)
	if err != nil {
		return nil, err
	}
	return json.Marshal(p)
}

// toAdmissionErrResponse is a helper function to create an AdmissionResponse
// with an embedded error
func toAdmissionErrResponse(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metaV1.Status{
			Message: err.Error(),
		},
	}
}
