#!/usr/bin/env bash

set -euxo pipefail

minikube start --driver=virtualbox --insecure-registry "10.0.0.0/8"
minikube addons enable registry

docker start minikube-registry || docker run --name=minikube-registry --rm -ti -d --network=host alpine /bin/ash -c "apk add socat && socat TCP-LISTEN:5000,reuseaddr,fork TCP:$(minikube ip):5000"



