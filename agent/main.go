package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jayce-jia/tidb-latency-agent-example/agent/executor"
	"k8s.io/klog"
)

type LatencyChangeHandler struct {
	configHolder *executor.ConfigHolder
}

func (h *LatencyChangeHandler) handleLatencyChange(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	latStr := vars["latency"]
	if latency, err := time.ParseDuration(latStr); err != nil {
		klog.Error(err)
		w.Write([]byte("Invalid Latency: " + latStr))
	} else {
		h.configHolder.Latency = latency
	}
}

func main() {
	var mgntPort int
	var initLatency, applyPeriod time.Duration
	flag.IntVar(&mgntPort, "port", 2332, "Agent Management Port.")
	flag.DurationVar(&initLatency, "latency", 0, "Initial Latency For The Agent.")
	flag.DurationVar(&applyPeriod, "period", 1*time.Second, "The Period Applying Latency To The Pod.")
	flag.Parse()

	configHolder := &executor.ConfigHolder{Latency: 0, ApplyPeriod: applyPeriod}
	executor := &executor.TcExecutor{ConfigHolder: configHolder}
	latencyChangeHandler := LatencyChangeHandler{configHolder: configHolder}

	// start the deamon to set the latency
	executor.Init()
	router := mux.NewRouter()
	subRouter := router.PathPrefix("/latency").Subrouter()
	// endpoint to set the latency
	subRouter.HandleFunc("/{latency}", latencyChangeHandler.handleLatencyChange)
	// endpoint to query the current latency
	subRouter.HandleFunc("", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte(configHolder.Latency.String()))
	})
	// endpoint for health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {})
	http.Handle("/", router)
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", mgntPort),
	}
	server.ListenAndServe()
}
