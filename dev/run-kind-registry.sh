#!/bin/sh
set -o errexit

docker start kind-registry || docker run -d -p "127.0.0.1:5000:5000" --name "kind-registry" registry:2

docker network connect "kind" "kind-registry" || true

