#!/usr/bin/env bash

set -euxo pipefail

minikube start --driver=virtualbox --insecure-registry "10.0.0.0/8"
minikube addons enable registry

docker start minikube-registry || docker run --name=minikube-registry --rm -ti -d --network=host alpine /bin/ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"

helm repo add falcosecurity https://falcosecurity.github.io/charts
helm repo update

# create namespace idempotently
kubectl create namespace falco --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
kubectl wait --for=condition=ready pods --all -n cert-manager --timeout=3m || true
sleep 10


__dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
kubectl apply -f "$__dir/k8s" -R

helm upgrade -i falco falcosecurity/falco -n falco -f "$__dir/falco-values.yaml" -f "$__dir/custom-rules.yaml"


