```bash
kubectl create namespace tarian-system

# install dependency: postgresql
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install tarian-postgresql bitnami/postgresql -n tarian-system --set postgresqlUsername=postgres --set postgresqlPassword=tarian --set postgresqlDatabase=tarian

helm install tarian-server ./charts/tarian-server -n tarian-system
helm install tarian-cluster-agent ./charts/tarian-cluster-agent/ -n tarian-system
```

