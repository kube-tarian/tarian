#!/usr/bin/env bash

set -euxo pipefail

export TARIAN_SERVER_ADDRESS=localhost:31051
export PATH=$PATH:./bin

function retry {
  local retries=$1
  shift

  local count=0
  until "$@"; do
    exit=$?
    wait=1
    count=$(($count + 1))
    if [ $count -lt $retries ]; then
      echo "Retry $count/$retries exited $exit, retrying in $wait seconds..."
      sleep $wait
    else
      echo "Retry $count/$retries exited $exit, no more retries left."
      return $exit
    fi
  done
  return 0
}


# run db migration and seed data
retry 10 kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server dgraph apply-schema

tarianctl add constraint --name nginx --namespace default --match-labels run=nginx --allowed-processes=pause,tarian-pod-agent,nginx 
tarianctl add constraint --name nginx-files --namespace default --match-labels run=nginx --allowed-file-sha256sums=/usr/share/nginx/html/index.html=38ffd4972ae513a0c79a8be4573403edcd709f0f572105362b08ff50cf6de521
tarianctl get constraints
kubectl get pods -n tarian-system

kubectl logs deploy/tarian-controller-manager -n tarian-system

sleep 10s
kubectl get MutatingWebhookConfiguration -o yaml

# test pod-agent injection
# simulate the monitored file content changed
sed -i 's/Welcome/Welcome-updated/g' dev/config/monitored-pod/configmap.yaml
kubectl apply -f dev/config/monitored-pod -R
kubectl get pods
retry 3 kubectl wait --for=condition=ready pod/nginx --timeout=1m

echo test $(kubectl get pod nginx -o json | jq -r '.spec.containers | length') -eq 2 || (echo "expected container count 2" && false)
test $(kubectl get pod nginx -o json | jq -r '.spec.containers | length') -eq 2 || (echo "expected container count 2" && false)

# simulate unknown process runs
kubectl exec -ti nginx -c nginx -- sleep 15

# output for debugging
kubectl logs nats-0 -n tarian-system
kubectl logs deploy/tarian-server -p -n tarian-system || true
kubectl logs deploy/tarian-server -n tarian-system
tarianctl get events

# need to support both dev/config and charts, there's a naming difference
kubectl logs deploy/tarian-cluster-agent -n tarian-system

# assert contains sleep
tarianctl get events | grep sleep

# assert contains index.html
tarianctl get events | grep index.html

# output for debugging
kubectl run -ti --restart=Never get-alerts2 --image=curlimages/curl -- http://alertmanager.tarian-system.svc:9093/api/v2/alerts

# assert alerts were sent
echo $'test $(kubectl run -ti --restart=Never verify-alerts --image=curlimages/curl -- http://alertmanager.tarian-system.svc:9093/api/v2/alerts | jq \'. | length\') -gt 1' \
  $'|| (echo "expected alerts created" && false)'

test $(kubectl run -ti --restart=Never verify-alerts --image=curlimages/curl -- http://alertmanager.tarian-system.svc:9093/api/v2/alerts | jq '. | length') -gt 1 \
  || (echo "expected alerts created" && false)

# run command to register constraints
# multiple times to compensate occassional eBPF missing events
kubectl exec -ti nginx2 -c nginx -- bash -c 'for i in {1..200}; do zgrep; sleep 0.1s; done;'

# give time for tarian-cluser-agent to process data from node agents,
# due to many events generated from tarian-detector
sleep 5

tarianctl get constraints

# test register constraint using annotation
tarianctl get constraints | grep run=nginx2 | grep zgrep

# action
tarianctl add action --name nginx-delete --match-labels=run=nginx --action=delete-pod

tarianctl get actions

# expect the pod to be killed
kubectl exec -ti nginx -c nginx -- sleep 15 || true

kubectl get pods

# wait for terminating pod to be completely deleted
sleep 30s
kubectl get pods

echo $'(kubectl get pods -o  json | jq \'.items | any(.metadata.name=="nginx")\') == "false"'
test $(kubectl get pods -o  json | jq '.items | any(.metadata.name=="nginx")') == "false"
