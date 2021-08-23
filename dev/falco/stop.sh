#!/usr/bin/env bash

set -euxo pipefail

docker stop minikube-registry || true
minikube delete

