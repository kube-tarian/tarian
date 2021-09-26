When Falco integration is enabled, Tarian cluster agent subscribes Falco alerts via gRPC API.

## Setup

Falco support running gRPC API with mandatory mutual TLS (mTLS). So, firstly we need to prepare the certificates.

### Prepare Namespaces

```bash
kubectl create namespace tarian-system
kubectl create namespace falco
```

### Prepare Certificates for mTLS

You can setup certificates manually and save those certs to secrets accessible from Falco and Tarian pods.
For convenient, you can use [Cert Manager](https://cert-manager.io/) to manage the certs.

1. Install Cert Manager by following the guide https://cert-manager.io/docs/installation/

2. Setup certs

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned-issuer
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: root-ca
  namespace: cert-manager
spec:
  isCA: true
  commonName: root-ca
  secretName: root-secret
  privateKey:
    algorithm: ECDSA
    size: 256
  issuerRef:
    name: selfsigned-issuer
    kind: ClusterIssuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: ca-issuer
spec:
  ca:
    secretName: root-secret
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: falco-grpc-server
  namespace: falco
spec:
  isCA: false
  commonName: falco-grpc
  dnsNames:
  - falco-grpc.falco.svc
  - falco-grpc
  secretName: falco-grpc-server-cert
  usages:
  - server auth
  privateKey:
    algorithm: ECDSA
    size: 256
  issuerRef:
    name: ca-issuer
    kind: ClusterIssuer
    group: cert-manager.io
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: falco-integration-cert
  namespace: tarian-system
spec:
  isCA: false
  commonName: tarian-falco-integration
  dnsNames:
  - tarian-falco-integration
  usages:
  - client auth
  secretName: tarian-falco-integration
  privateKey:
    algorithm: ECDSA
    size: 256
  issuerRef:
    name: ca-issuer
    kind: ClusterIssuer
    group: cert-manager.io
```


### Install Falco

Save this to `falco-values.yaml`

```yaml
extraVolumes:
- name: grpc-cert
  secret:
    secretName: falco-grpc-server-cert
extraVolumeMounts:
- name: grpc-cert
  mountPath: /etc/falco/grpc-cert
falco:
  grpc:
    enabled: true
    unixSocketPath: ""
    threadiness: 1
    listenPort: 5060
    privateKey: /etc/falco/grpc-cert/tls.key
    certChain: /etc/falco/grpc-cert/tls.crt
    rootCerts: /etc/falco/grpc-cert/ca.crt
  grpcOutput:
    enabled: true
```

Then install Falco using Helm:

```bash
helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update

helm upgrade -i falco falcosecurity/falco -n falco -f falco-values.yaml
```


### Install Tarian server and Tarian cluster agent

```bash
helm repo add tarian https://kube-tarian.github.io/tarian
helm repo update

helm install tarian-postgresql bitnami/postgresql -n tarian-system \
  --set postgresqlUsername=postgres \
  --set postgresqlPassword=tarian \
  --set postgresqlDatabase=tarian

helm upgrade -i tarian-server tarian/tarian-server --devel -n tarian-system

kubectl wait --for=condition=ready pod --all -n tarian-system
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server db migrate

helm upgrade -i tarian-cluster-agent tarian/tarian-cluster-agent --devel -n tarian-system \
  --set clusterAgent.falco.clientTlsSecretName=tarian-falco-integration \
  --set clusterAgent.falco.grpcServerHostname=falco-grpc.falco.svc \
  --set clusterAgent.falco.grpcServerPort=5060
```


### Verifying

After the above step, you should see falco alert in `tarianctl get events` (See how to use tarianctl in the docs).

