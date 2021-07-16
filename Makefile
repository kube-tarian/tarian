.PHONY: default
default:
	CGO_ENABLED=0 go build -a -o ./bin/tarian-server ./cmd/tarian-server/
	CGO_ENABLED=0 go build -a -o ./bin/tarian-cluster-agent ./cmd/tarian-cluster-agent/
