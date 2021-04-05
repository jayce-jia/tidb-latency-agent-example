package main

import (
	"crypto/tls"
	"flag"
	"net/http"
	"time"

	webhook "github.com/jayce-jia/tidb-latency-agent-example/inject/webhook"
	"k8s.io/klog"
)

func main() {
	var sidecarSpecFile, tlsCertFile, tlsKeyFile string
	flag.StringVar(&sidecarSpecFile, "sidecarSpecFile", "./inject/spec/sidecar.yaml", "Latency Agent Sidecar Spec File Location")
	flag.StringVar(&tlsCertFile, "tlsCertFile", "./server.crt", "TLS Cert File Location")
	flag.StringVar(&tlsKeyFile, "tlsKeyFile", "./server-key.pem", "TLS Key File Location")

	var agentContainerName, agentImage, agentImageTag string
	var agentManagementPort int
	var agentInitLatency, agentApplyPeriod time.Duration
	flag.StringVar(&agentContainerName, "agentContainerName", "latency-agent", "Latency Agent Sidecar Container Name")
	flag.StringVar(&agentImage, "agentImage", "github.com/jayce-jia/tidb-latency-agent-example", "Latency Agent Sidecar Image")
	flag.StringVar(&agentImageTag, "agentImageTag", "latest", "Latency Agent Sidecar Image Tag")
	flag.IntVar(&agentManagementPort, "agentManagementPort", 2332, "Latency Agent Sidecar Management Port")
	flag.DurationVar(&agentInitLatency, "agentInitLatency", 0, "Latency Agent Sidecar Initial Latency")
	flag.DurationVar(&agentApplyPeriod, "agentApplyPeriod", 1*time.Second, "Latency Agent Sidecar Apply Period")
	flag.Parse()

	pair, err := tls.LoadX509KeyPair(tlsCertFile, tlsKeyFile)
	if err != nil {
		klog.Errorf("Error Loading TLS Certificate[%s|%s], please check your configurations.\n", tlsCertFile, tlsKeyFile, err)
		panic(err)

	}
	server := &http.Server{
		Addr:      ":443",
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
	}

	config := webhook.AgentConfig{
		ContainerName:  agentContainerName,
		Image:          agentImage,
		ImageTag:       agentImageTag,
		ManagenemtPort: int32(agentManagementPort),
		InitLatency:    agentInitLatency,
		ApplyPeriod:    agentApplyPeriod,
	}
	wh := webhook.NewWebhook(config)

	http.DefaultServeMux.HandleFunc("/inject", wh.ServeInject)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {})
	klog.Infoln("Starting the Webhook TLS Server")
	if err := server.ListenAndServeTLS("", ""); err != nil {
		klog.Errorln("Server failed to start.", err)
		panic(err)
	}
}
