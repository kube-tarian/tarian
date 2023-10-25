package testing

import (
	"net"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

var grpcSrv *grpc.Server

// StartFakeServer starts a fake gRPC server.
func StartFakeServer(t *testing.T, serverAddr string) {
	lis, err := net.Listen("tcp", serverAddr)
	if err != nil {
		assert.NoError(t, err)
	}

	grpcSrv = grpc.NewServer()

	if err = grpcSrv.Serve(lis); err != nil {
		assert.NoError(t, err)
	}
}

// CloseFakeServer closes a fake gRPC server.
func CloseFakeServer() {
	grpcSrv.Stop()
}

// LogOutputWriter is a writer for log output.
type LogOutputWriter struct {
	// Output is the log output.
	Output *[]byte
}

// Write writes the log output.
func (w *LogOutputWriter) Write(p []byte) (n int, err error) {
	*w.Output = append(*w.Output, p...)
	return len(p), nil
}

// CleanLog cleans the log output.
func CleanLog(input string) string {
	splitLog := strings.Split(input, "msg=")
	if len(splitLog) > 2 {
		re := regexp.MustCompile(`level=.*?msg=`)
		input = re.ReplaceAllString(input, "")
		// re = regexp.MustCompile(`msg=`)
		// input = re.ReplaceAllString(input, "")
	} else {
		input = splitLog[len(splitLog)-1]
	}
	spaceRe := regexp.MustCompile(`\s+`)
	input = spaceRe.ReplaceAllString(input, " ")

	newlineRe := regexp.MustCompile(`\n+`)
	input = newlineRe.ReplaceAllString(input, "\n")
	return strings.TrimSpace(input)
}
