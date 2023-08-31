package main

import (
	ver "github.com/kube-tarian/tarian/cmd"
	"github.com/kube-tarian/tarian/cmd/tarian-cluster-agent/cmd"
)

var (
	version    = "dev"
	commit     = "main"
	versionStr = version + " (" + commit + ")"
)

func main() {
	ver.SetVersion(versionStr)
	cmd.Execute()
}
