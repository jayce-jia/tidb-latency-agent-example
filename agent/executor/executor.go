package executor

import (
	"os/exec"
	"strings"
	"time"

	"k8s.io/klog"
)

type TcExecutor struct {
	ConfigHolder *ConfigHolder
}

func (e *TcExecutor) Init() {
	err := e.applyLatencyIfNecessary()
	if err != nil {
		klog.Error("Failed to initialize TcExecutor", err)
		panic(err)
	}
	go e.createDaemon()
}

func (e *TcExecutor) createDaemon() {
	klog.Info("Tc Executor Daemon Started.")
	for {
		err := e.applyLatencyIfNecessary()
		if err != nil {
			klog.Error("Error when applying latency", err)
		}
		<-time.After(e.ConfigHolder.ApplyPeriod)
	}
}

// apply latency if necessary
// if latency didn't change, then do nothing
// if latency changed, apply the latency using tc
func (e *TcExecutor) applyLatencyIfNecessary() error {
	queryOutBytes, err := exec.Command("tc", "qdisc", "show", "dev", "eth0").Output()
	if err != nil {
		return err
	}
	queryOut := string(queryOutBytes)
	delayIdx := strings.Index(string(queryOutBytes), "delay")

	var currentLatency time.Duration
	if delayIdx >= 0 {
		currentLatency, _ = time.ParseDuration(queryOut[delayIdx+6:])
	} else {
		currentLatency = 0
	}

	if int64(currentLatency) != int64(e.ConfigHolder.Latency) {
		_, err := exec.Command("tc", "qdisc", "replace", "dev", "eth0", "root", "netem", "delay", e.ConfigHolder.Latency.String()).Output()
		if err != nil {
			return err
		}
	}

	return nil

}
