<p align="center"><img src="logo/tarian-new-logo-1.png" width="175"></p>

# Tarian

Protect your applications running on Kubernetes from malicious attacks by pre-registering your trusted processes and trusted file signatures. Tarian will detect unknown processes and changes to the registered files, then it will send alerts and take an automated action. Save your K8s environment from Ransomware!

We want to maintain this as an open-source project to fight against the attacks on our favorite Kubernetes ecosystem. By continuous contribution, we can fight threats together as a community.

[![Build status](https://img.shields.io/github/workflow/status/kube-tarian/tarian/CI?style=flat)](https://github.com/kube-tarian/tarian/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/kube-tarian/tarian)](https://goreportcard.com/report/github.com/kube-tarian/tarian)
[![codecov](https://codecov.io/gh/kube-tarian/tarian/graph/badge.svg?token=PH8E9ZOVR4)](https://codecov.io/gh/kube-tarian/tarian)

---

**How does Tarian work?**

Tarian Cluster Agent runs in Kubernetes cluster detecting unknown processes and unknown changes to files, report them to Tarian Server, and optionally take action: delete the violated pod. It uses eBPF to detect new processes. For file change detection, Tarian Cluster Agent injects a sidecar container in your main application's pod which will check file checksums in the configured path and compare them with the registered checksums in Tarian Server. Tarian will be a part of your Application's pod from dev to prod environment, hence you can register to your Tarian DB what is supposed to be happening & running in your container + file signatures to be watched + what can be notified + action to take (self destroy the pod) based on changes detected. Shift-left your detection mechanism!


**What if an unknown change happens inside the container which is not in Tarian's registration DB, how does Tarian react to it?**

If an unknown change happens, Tarian can simply notify observed analytics to your Security Team. Then your Security Engineers can register that change in Tarian DB whether it's considered a threat or not. Also, based on their analysis they can configure what action to take when that change happens again.


**How does the contribution of community helps to fight against the threats via Tarian?**

Any new detection analyzed & marked as a threat by your Security Experts, if they choose, can be shared to the open-source Tarian community DB with all the logs, strings to look for, observation, transparency, actions to configure, ... Basically anything the Experts want to warn about & share with the community. You can use that information as a Tarian user and configure actions in the Tarian app which is used in your environment. This is basically a mechanism to share info about threats & what to do with them. This helps everyone using Tarian to take actions together in their respective K8s environments by sharing their knowledge & experience.


**What kind of action(s) would Tarian take based on known threat(s)?**

Tarian would simply self destroy the pod it's running on. If the malware/virus spreads to the rest of the environment, well you know what happens. So, Tarian is basically designed to help reduce the risk as much as possible by destroying pods. Provisioning a new pod will be taken care of by K8s deployment. Tarian will only do destruction of the pods only if you tell Tarian to do so. If you don't want any actions to happen, you don't have to configure or trigger any; you can simply tell Tarian to just notify you. Tarian basically does what you want to be done to reduce the risk.


**Why another new security tool when there are many tools available already, like Falco, Kube-Hunter, Kube-Bench, Calico Enterprise Security, and many more security tools (open-source & commercial) that can detect & prevent threats at network, infra & application level? Why Tarian?**

The main reason Tarian was born is to fight against threats in Kubernetes together as a community. Another reason was, what if there is still some sophisticated attack which is capable of penetrating every layer of your security, able to reach your runtime app (Remote Code Execution) and your storage volumes, and capable of spreading to damage or lock your infra & data?! What do you want to do about such attacks, especially which turns into ransomware. Tarian is designed to reduce such risks, by taking action(s). We know that Tarian is not the ultimate solution, but we are confident that it can help reduce risks especially when knowledge is shared continuously by the community. From a technical perspective, Tarian can help reduce the risk by destroying the infected resources.

## Architecture diagram

![Arch. Diagram](./docs/architecture-diagram.png)

## Requirements

- Supported Kubernetes version (currently 1.22+)
- Kernel version >= 5.8
- Kernel with [BTF](https://www.kernel.org/doc/html/latest/bpf/btf.html) information to support eBPF CO-RE.
  Some major Linux distributions come with kernel BTF already built in. If your kernel doesn't come with BTF built-in,
  you'll need to build custom kernel. See [BPF CO-RE](https://github.com/libbpf/libbpf#bpf-co-re-compile-once--run-everywhere).


### Tested on popular Kubernetes Environments/Services:

| Environment                                  | Working            | Notes                                                              |
|----------------------------------------------|--------------------|--------------------------------------------------------------------|
| Kind v0.14.0                                 | :heavy_check_mark: |                                                                    |
| Minikube v1.26.0                             | :heavy_check_mark: |                                                                    |
| Linode Kubernetes Engine (LKE) 1.22          | :heavy_check_mark: |                                                                    |
| Digital Ocean Kubernetes Engine (DOKS) 1.22  | :heavy_check_mark: |                                                                    |
| Google Kubernetes Engine (GKE) 1.22          | :heavy_check_mark: |                                                                    |
| Amazon Elastic Kubernetes Engine (EKS)       | :heavy_minus_sign: | [kernel < 5.8](https://github.com/awslabs/amazon-eks-ami/pull/862) |
| Azure Kubernetes Service (AKS)               | :heavy_minus_sign: | [kernel < 5.8](https://github.com/Azure/AKS/issues/2883)           |


### Prepare Namespaces

```bash
kubectl create namespace tarian-system
```

### Setup Dgraph Database

You can use any [Dgraph installation](https://dgraph.io/docs/deploy/kubernetes/) option as long as it can be accessed from the tarian server.


### Install tarian

1. Install tarian using Helm

```bash
helm repo add tarian https://kube-tarian.github.io/helm-charts
helm repo update

helm upgrade -i tarian-server tarian/tarian-server --devel -n tarian-system --set server.dgraph.address=DGRAPH_ADDRESS:PORT
helm upgrade -i tarian-cluster-agent tarian/tarian-cluster-agent --devel -n tarian-system
```

2. Wait for all the pods to be ready

```bash
kubectl wait --for=condition=ready pod --all -n tarian-system
```

3. Apply Dgraph schema

```bash
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server dgraph apply-schema
```
### Install tarian using tarianctl cli
Download tarianctl bin from github release page.

Run:
```
tarianctl install
```

You can use following flags to customize your installation.

```
Install Tarian on Kubernetes.

Usage:
  tarianctl install [flags]

Flags:
      --agents-values strings   Path to the helm values file for Tarian Cluster Agent and Node agent .
      --charts string           Path to the tarian helm charts directory.
      --dgraph-values strings   Path to the helm values file for DGraph.
  -h, --help                    help for install
  -n, --namespace string        Namespace to install Tarian. (default "tarian-system")
      --nats-values strings     Path to the helm values file for Nats.
      --server-values strings   Path to the helm values file for Tarian Server.

Global Flags:
  -k, --kubeconfig string                 path to the kubeconfig file to use
  -e, --log-formatter string              valid log formatters: json, text(default) (default "text")
  -l, --log-level string                  valid log levels: debug, info(default), warn/warning, error, fatal (default "info")
  -s, --server-address string             tarian server address to communicate with (default "localhost:50051")
  -c, --server-tls-ca-file string         ca file that server uses for TLS connection
  -t, --server-tls-enabled                if enabled, it will communicate with the server using TLS
  -i, --server-tls-insecure-skip-verify   if set to true, it will skip server's certificate chain and hostname verification (default true)

```
## Configuration

See helm chart values for
- [tarian-server](https://github.com/kube-tarian/tarian/blob/main/charts/tarian-server/values.yaml)
- [tarian-cluster-agent](https://github.com/kube-tarian/tarian/blob/main/charts/tarian-cluster-agent/values.yaml)


## Cloud / Vendor specific configuration

### Private GKE cluster

Private GKE cluster by default creates firewall rules to restrict master to nodes communication only on ports `443` and `10250`.
To inject tarian-pod-agent container, tarian uses a mutating admission webhook. The webhook server runs on port `9443`. So, we need
to create a new firewall rule to allow ingress from master IP address range to nodes on tcp port **9443**.

For more details, see GKE docs on this topic: [https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#add_firewall_rules](https://cloud.google.com/kubernetes-engine/docs/how-to/private-clusters#add_firewall_rules).


## Usage

### Use tarianctl to control tarian-server

1. Download from Github [release page](https://github.com/kube-tarian/tarian/releases)
2. Extract the file and copy tarianctl to your PATH directory
3. Expose tarian-server to your machine, through Ingress or port-forward. For this example, we'll use port-forward:

```bash
kubectl port-forward svc/tarian-server -n tarian-system 41051:80
```

4. Configure server address with env var

```
export TARIAN_SERVER_ADDRESS=localhost:41051
```

### To see violation events

```bash
tarianctl get events
```

### Add a process constraint

```bash
tarianctl add constraint --name nginx --namespace default \
  --match-labels run=nginx \
  --allowed-processes=pause,tarian-pod-agent,nginx 
```

```bash
tarianctl get constraints
```

### Add a file constraint

```bash
tarianctl add constraint --name nginx-files --namespace default \
  --match-labels run=nginx \
  --allowed-file-sha256sums=/usr/share/nginx/html/index.html=38ffd4972ae513a0c79a8be4573403edcd709f0f572105362b08ff50cf6de521
```

```bash
tarianctl get constraints
```

### Run tarian agent in a pod

Then after the constraints are created, we inject tarian-pod-agent to the pod by adding an annotation:

```yaml
metadata:
  annotations:
    pod-agent.k8s.tarian.dev/threat-scan: "true"
```

Pod with this annotation will have an additional container injected (tarian-pod-agent). The tarian-pod-agent container will 
continuously verify the runtime environment based on the registered constraints. Any violation would be reported, which would be
accessible with `tarianctl get events`.


### Demo: Try a pod that violates the constraints

```bash
kubectl apply -f https://raw.githubusercontent.com/kube-tarian/tarian/main/dev/config/monitored-pod/configmap.yaml
kubectl apply -f https://raw.githubusercontent.com/kube-tarian/tarian/main/dev/config/monitored-pod/pod.yaml

# wait for it to become ready
kubectl wait --for=condition=ready pod nginx

# simulate unknown process runs
kubectl exec -ti nginx -c nginx -- sleep 15

# you should see it reported in tarian
tarianctl get events
```

## Alert Manager Integration

Tarian comes with Prometheus Alert Manager by default. If you want to use another alert manager instance:

```bash
helm install tarian-server tarian/tarian-server --devel \
  --set server.alert.alertManagerAddress=http://alertmanager.monitoring.svc:9093 \
  --set alertManager.install=false \
  -n tarian-system
```

To disable it, you can set the alertManagerAddress value to empty.

## Troubleshooting

See [docs/troubleshooting.md](docs/troubleshooting.md)

## Automatic Constraint Registration

When tarian-pod-agent runs in registration mode, instead of reporting unknown processes and files as violations, it automatically registers them as a new constraint. This is convenient to save time from registering manually.

To enable constraint registration, the cluster-agent needs to be configured.

```bash
helm install tarian-cluster-agent tarian/tarian-cluster-agent --devel -n tarian-system \
  --set clusterAgent.enableAddConstraint=true
```

```yaml
metadata:
  annotations:
    # register both processes and file checksums
    pod-agent.k8s.tarian.dev/register: "processes,files"
    # ignore specific paths from automatic registration
    pod-agent.k8s.tarian.dev/register-file-ignore-paths: "/usr/share/nginx/**/*.txt"
```

Automatic constraint registration can also be done in a dev/staging cluster, so that there would be less changes in production.

## Other supported annotations

```yaml
metadata:
  annotations:
    # specify how often tarian-pod-agent should verify file checksum
    pod-agent.k8s.tarian.dev/file-validation-interval: "1m"
```

## Securing tarian-server with TLS

To secure tarian-server with TLS, create a secret containing the TLS certificate. You can create the secret manually,
or using [Cert Manager](https://cert-manager.io/). Once you have the secret, you can pass the name to the helm chart value:

```
helm upgrade -i tarian-server tarian/tarian-server --devel -n tarian-system \
  --set server.tlsSecretName=tarian-server-tls
```

## Contributing

See [docs/contributing.md](docs/contributing.md)

## Code of Conduct
See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)

## CodeOwners & Maintainers list
See [MAINTAINERS.md](MAINTAINERS.md)

## Join our Slack channel " tarian "
[Kube-Tarian-Slack](https://join.slack.com/t/kube-tarian/shared_invite/zt-118iqu4g6-wopEIyjqD_uy5uXRDChaLA)
