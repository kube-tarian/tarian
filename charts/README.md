Install dgraph:

- https://dgraph.io/docs/deploy/kubernetes/

Then:

```bash
kubectl create namespace tarian-system

helm install tarian-server ./charts/tarian-server -n tarian-system --set server.dgraph.address=DGRAPH_ADDRESS:PORT
helm install tarian-cluster-agent ./charts/tarian-cluster-agent/ -n tarian-system
```

