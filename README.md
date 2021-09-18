# Tarian

> ###### We want to maintain this as open-source to fight against the attacks on our favorite Kubernetes ecosystem. We can fight threats together as a community, by continuous contribution.   


##### Protect your Applications running on Kubernetes from malicious attacks by pre-registering your source code signatures, runtime processes monitoring, runtime source code monitoring, change detection, alerting, pre-configured & instant respond actions based on detections and also sharing detections with community. Save your K8s environment from Ransomware! 

#

[![Build status](https://img.shields.io/github/workflow/status/devopstoday11/tarian/CI?style=flat)](https://github.com/devopstoday11/tarian/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/devopstoday11/tarian)](https://goreportcard.com/report/github.com/devopstoday11/tarian)

#

How does Tarian works?
> Tarian runs as sidecar container in your main application's pod watching for changes in process IDs, number of processes running, parent & child processes relation changes, files & directories in the file system which belongs to your application, changes happened to your files signatures, etc. Tarian will be part of your Application's pod from dev to prod environment, hence you can register to your Tarian DB what is supposed to happening & running in your container + what can be watched + what can be notified + action (self destroy the pod) to take based on changes detected. Shift-left your detection mechanism! 

What if unknow change happens inside the container which is not in Tarian's registration DB, how does Tarian react to such? 
> If an unknown change happens Tarian can simply notify observed analytics to your Security Team + send the log to the Security Team. Then your Security Engineers can register that change in Tarian DB as whether it's a threat or not & based on their analysis they can configure what action to take. That action will be sent as command to the sidecar Tarian app to perform the action. 

How does the contribution of community helps to fight against the threats via Tarian?
> Any new detection analyzed & marked as threat by your Security Experts, if they choose, can be shared to the open-source Tarian community DB with all the logs, strings to look for, observation, transparency, actions to configure, basically anything the Expert wants to warn & share with the community. You can use that information as Tarian user and  configure actions in your Tarian app which you use in your environment. This is basically sharing the info about threats & what to do about them. This helps everyone using Tarian to share their analysis & take actions together in their respective K8s environments by sharing the knowledge & experience. 

What kind of action(s) would Tarian take based on known threat(s)?
> Tarian would simply self destroy the pod it's running on along with deleting any files on the volumes to reduce the risk. If the malware/virus spreads to rest of the environment, well you know what happens. So, Tarian is basically designed to help reduce the risk as much as possible, by destroying pods. Provisioning new pod will be taken care by K8s since that's how K8s works. Tarian will only do destruction of pods, and only if you tell Tarian to do so by pre-configuring the action in the Tarian controller or by telling Tarian to do so on-the-fly. If you don't want any actions to happen, you don't have to configure or trigger any; you can simply tell Tarian to just notify you. Tarian basically does what you want to be done to reduce the risk. 

#

Why another new security tool when there are many tools available already, like Falco, Kube-Hunter, Kube-Bench, Calico Enterprise Security, and many more security tools (open-source & commercial) that can detect & prevent from threats at network level, infra level & application level? Why Tarian?
> Like I mentioned above, the main reason Tarian was born is to fight together as a community against threats in Kubernetes. And another reason was, what if there is still some sophisticated attack which is capable of penetrating every layer of your securities & able to reach your runtime app, your storage volumes and which is capable to spreading to damage or lock your infra & data?! What do you want to do about such attacks, especially which turns into ransomeware. Tarian is designed to reduce such risks, by taking action(s). We know that Tarian is not ultimate solution, but we are confident that it can help reduce risks especially when knowledge is shared by community continuously and also from technical perspective Tarian can help reduce risk by destroying the infected resources. 

#

#### Architecture diagram
> I will finalize the design diagram soon once I finish talking to some security experts (I already talked to few, and have few more discussions pending). 

![Arch. Diagram](https://github.com/devopstoday11/tarian/blob/5eeed9a0bd5875e6cee423d2d12161a3f7d2d84c/Kube-Tarian.svg)

#

## Install

1. Create tarian-system namespace

```bash
kubectl create namespace tarian-system
```

2. Prepare a Postgresql Database. You can use a DB as a service from your Cloud Services or you can also run by yourself in the cluster. For example to install the DB in the cluster, run:

```bash
helm install tarian-postgresql bitnami/postgresql -n tarian-system --set postgresqlUsername=postgres --set postgresqlPassword=tarian --set postgresqlDatabase=tarian
```

3. Install tarian

```bash
helm repo add tarian https://devopstoday11.github.io/tarian
helm repo update

helm install tarian-server tarian/tarian-server --devel -n tarian-system
helm install tarian-cluster-agent tarian/tarian-cluster-agent --devel -n tarian-system
```

4. Wait for all pods to be ready

```bash
kubectl wait --for=condition=ready pod --all -n tarian-system
```

5. Run database migration to create the required tables

```bash
kubectl exec -ti deploy/tarian-server -n tarian-system -- ./tarian-server db migrate
```

## Configuration

See helm chart values for
- [tarian-server](https://github.com/devopstoday11/tarian/blob/main/charts/tarian-server/values.yaml)
- [tarian-cluster-agent](https://github.com/devopstoday11/tarian/blob/main/charts/tarian-cluster-agent/values.yaml)

## Usage

### Use tarianctl to control tarian-server

1. Download from Github [release page](https://github.com/devopstoday11/tarian/releases/download/v0.0.3-alpha3/tarian_0.0.3-alpha3_linux_amd64.tar.gz)
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
tarianctl add constraint --name nginx --namespace default --match-labels run=nginx --allowed-processes=pause,tarian-pod-agent,nginx 
```

```bash
tarianctl get constraints
```

### Add a file constraint

```bash
tarianctl add constraint --name nginx-files --namespace default --match-labels run=nginx --allowed-file-sha256sums=/usr/share/nginx/html/index.html=38ffd4972ae513a0c79a8be4573403edcd709f0f572105362b08ff50cf6de521
```

```bash
tarianctl get constraints
```

### Try a pod that violates the constraints

```bash
kubectl apply -f https://raw.githubusercontent.com/devopstoday11/tarian/main/dev/config/monitored-pod/configmap.yaml
kubectl apply -f https://raw.githubusercontent.com/devopstoday11/tarian/main/dev/config/monitored-pod/pod.yaml

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

## Falco Integration

See [docs/falco-integration.md](docs/falco-integration.md)

