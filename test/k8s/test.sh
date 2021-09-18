#!/usr/bin/env bash

set -euxo pipefail

export TARIAN_SERVER_ADDRESS=localhost:31051
export PATH=$PATH:./bin

# run db migration and seed data
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server db migrate
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server dev seed-data
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
kubectl wait --for=condition=ready pod/nginx --timeout=5m

echo test $(kubectl get pod nginx -o json | jq -r '.spec.containers | length') -eq 2 || (echo "expected container count 2" && false)
test $(kubectl get pod nginx -o json | jq -r '.spec.containers | length') -eq 2 || (echo "expected container count 2" && false)

# simulate unknown process runs
kubectl exec -ti nginx -c nginx -- sleep 15

# output for debugging
tarianctl get events

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

tarianctl get constraints

# test register constraint using annotation
tarianctl get constraints | grep run=nginx2
