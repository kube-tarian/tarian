package main

import (
	ver "github.com/kube-tarian/tarian/cmd"
	"github.com/kube-tarian/tarian/cmd/tarian-server/cmd"
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
