#!/usr/bin/env bash

set -euxo pipefail

export TARIAN_SERVER_ADDRESS=localhost:31051
export PATH=$PATH:./bin

# run db migration and seed data
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server dgraph apply-schema || true || sleep 10
# retry 
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server dgraph apply-schema 

tarianctl add constraint --name nginx --namespace default --match-labels run=nginx --allowed-processes=pause,tarian-pod-agent,nginx 
tarianctl add constraint --name nginx-files --namespace default --match-labels run=nginx --allowed-file-sha256sums=/usr/share/nginx/html/index.html=38ffd4972ae513a0c79a8be4573403edcd709f0f572105362b08ff50cf6de521
tarianctl get constraints
kubectl get pods -n tarian-system

kubectl logs deploy/tarian-cluster-agent-controller-manager -n tarian-system

sleep 10s
kubectl get MutatingWebhookConfiguration -o yaml

# test pod-agent injection
# simulate the monitored file content changed
sed -i 's/Welcome/Welcome-updated/g' dev/config/monitored-pod/configmap.yaml
kubectl apply -f dev/config/monitored-pod -R
kubectl get pods
kubectl wait --for=condition=ready pod/nginx --timeout=1m || true
kubectl wait --for=condition=ready pod/nginx --timeout=1m || true
kubectl wait --for=condition=ready pod/nginx --timeout=1m

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
kubectl logs `kubectl get ds -n tarian-system -o name` -n tarian-system
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
kubectl exec -ti nginx2 -c nginx -- pwd
kubectl exec -ti nginx2 -c nginx -- ls /

# give time for tarian-cluser-agent to process data from node agents
sleep 15

tarianctl get constraints

# test register constraint using annotation
tarianctl get constraints | grep run=nginx2 | grep pwd

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
