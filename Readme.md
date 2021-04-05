# TL;DR

This project is an example project that a latency agent being automatically injected to the tidb pod as a sidecar.

# Features
## Agent

- Capability to add a configurable network latency to the host pod.
- Initial latency is configurable.
- The latency can be set via http management endpoint.
- The latency can be queried via http management endpoint.

## Auto Injection

- The Agent can be automatically injected to any newly created tidb pod.
- Auto-Injection can be enabled/disabled via labels on namespaces.

# Quick Start
## Installation
### Clone the project
```
git clone https://github.com/jayce-jia/tidb-latency-agent-example.git
cd tidb-latency-agent-example
```

### Build Agent
```
cd agent
make docker-build
```
The image will be built locally. You may push it to remote registry per your need.

### Install Injector Webhook
``` 
cd ../inject
sh ./deploy-webhook.sh
```

You may valid the installation by checking:
```
kubectl -n latency-agent-admin get all
```

And the successful result would be:
```
NAME                                                      READY   STATUS    RESTARTS   AGE
pod/sidecar-injector-webhook-deployment-5c8b8544d-25gc7   1/1     Running   0          2m4s

NAME                               TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/sidecar-injector-webhook   ClusterIP   10.102.228.60   <none>        443/TCP   2m4s

NAME                                                  READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/sidecar-injector-webhook-deployment   1/1     1            1           2m4s

NAME                                                            DESIRED   CURRENT   READY   AGE
replicaset.apps/sidecar-injector-webhook-deployment-5c8b8544d   1         1         1       2m4s
```

## Verify the Injection
### Install TiDB Operator
```
# Install TiDB Operator CRD
kubectl apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/v1.1.11/manifests/crd.yaml

# Install TiDB Operator via Helm
helm repo add pingcap https://charts.pingcap.org/
kubectl create namespace tidb-admin && helm install --namespace tidb-admin tidb-operator pingcap/tidb-operator --version v1.1.11
```
For more details, please refer the official [documentation](https://docs.pingcap.com/tidb-in-kubernetes/stable/get-started)

### Install TiDB Cluster
```
# Create a new namespace for tidb cluster
kubectl create ns tidb-cluster

# Tag the label to the namespace to enable auto-injection
# This is mandatory step!!!
kubectl label ns tidb-cluster jayce.jia.latency.agent.sidecar-injector=enabled

# Install TiDB Cluster
kubectl -n tidb-cluster apply -f https://raw.githubusercontent.com/pingcap/tidb-operator/master/examples/basic/tidb-cluster.yaml
```

### Verify the Injection
Wait until the TiDB cluster is up, then run the following command to verify.
```
kubectl -n tidb-cluster get pods -l app.kubernetes.io/component=tidb
```

And you may see:
```
NAME           READY   STATUS    RESTARTS   AGE
basic-tidb-0   3/3     Running   0          70s
```

And by the command:
```
kubectl -n tidb-cluster describe pod/basic-tidb-0
```
You may see a container named latency-agent there! Cheers!

## Verify the Agent

Note: The following scripts should run inside the cluster, so that the pod ip is accessible

### Get Pod IP
```
export pod_ip=$(kubectl -n tidb-cluster get pod/basic-tidb-0 -o jsonpath='{$.status.podIP}')
```

### Get Current Latency
```
curl http://${pod_ip}:2332/latency
```

### Set The Latency
```
curl -X GET http://${pod_ip}:2332/latency/2s
```

### Verify If The Latency Is Applied
```
ping ${pod_ip}
```

# How It Works
## Latency

We use linux network tool `tc` to apply the latency to the pod. See more details on the [website](https://wiki.linuxfoundation.org/networking/netem).

## Auto-Injection

Kubernetes provides `MutatingAdmissionWebhook` for resource mutations. Setting up the webhook and appending the agent container to the pod during the admission will do the work. See more details [here](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#mutatingadmissionwebhook)

# References
Kubernetes Official Document: [Link](https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers)
Kubernetes Webhook Example: [Link](ttps://github.com/kubernetes/kubernetes/blob/v1.13.0/test/images/webhook/main.go)
Istio Auto Injection Document: [Link](https://istio.io/latest/docs/setup/additional-setup/sidecar-injection/#automatic-sidecar-injection)
