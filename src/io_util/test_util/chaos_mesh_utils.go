package test_util

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

// WaitForChaosMeshRunning queries the NetworkChaos resources on the cluster and waits until the specified bandwidth limit
// exists between node and otherNode
func WaitForChaosMeshRunning(ctx context.Context, node string, otherNode string) {
	networkChaosName := fmt.Sprintf("%s-%s", node, otherNode)
	chaosSelectedArgs := fmt.Sprintf("kubectl describe networkchaos %s | grep -B 1 \"Type:.*Selected\" | awk '/Status:/ {print $2}'", networkChaosName)
	chaosInjectedArgs := fmt.Sprintf("kubectl describe networkchaos %s | grep -B 1 \"Type:.*AllInjected\" | awk '/Status:/ {print $2}'", networkChaosName)

	for {
		isSelectedCmd := exec.CommandContext(ctx, "bash", "-c", chaosSelectedArgs)
		isInjectedCmd := exec.CommandContext(ctx, "bash", "-c", chaosInjectedArgs)
		isSelectedOutput, err := isSelectedCmd.CombinedOutput()
		isSelected := strings.TrimSpace(string(isSelectedOutput)) == "True"
		fmt.Printf("Selected: %s\n", string(isSelectedOutput))
		handle(err)
		isInjectedOutput, err := isInjectedCmd.CombinedOutput()
		isInjected := strings.TrimSpace(string(isInjectedOutput)) == "True"
		fmt.Printf("Injected: %s\n", string(isInjectedOutput))
		handle(err)

		if isSelected && isInjected {
			break
		}

		// Delay between NetworkChaos queries
		time.Sleep(500 * time.Millisecond)
	}
}
