#! /bin/sh
export service="sidecar-injector-webhook"
export namespace="latency-agent-admin"

make docker-build

kubectl create namespace ${namespace} --dry-run=client -o yaml | kubectl apply -f -

cat ./spec/cert.yaml | kubectl -n ${namespace} apply -f -

cat ./spec/webhook.yaml | kubectl -n ${namespace} apply -f -