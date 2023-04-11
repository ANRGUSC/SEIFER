package inference_io

import (
	"context"
	"os"

	"github.com/Dat-Boi-Arjun/SEIFER/io_util/test_util"
)

const (
	ReadinessFile = "/readiness_check/ready.txt"
)

// inferencePodReadinessCheck is called after the inference sockets are set up, so the only thing it checks is whether the
// ChaosMesh bandwidth limit is active
func inferencePodReadinessCheck(ctx context.Context, node string, otherNode string) {
	test_util.WaitForChaosMeshRunning(ctx, node, otherNode)

	sendReadinessMessage()
}

func sendReadinessMessage() {
	message := []byte("ready")
	err := os.WriteFile(ReadinessFile, message, 0644)
	handle(err)
}
