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

	// In case the network chaos name is in this order of the node names
	networkChaosNameSwitched := fmt.Sprintf("%s-%s", otherNode, node)

	for {
		isSelected := getConditionStatus(ctx, networkChaosName, "Selected")
		switchedNameIsSelected := getConditionStatus(ctx, networkChaosNameSwitched, "Selected")

		isInjected := getConditionStatus(ctx, networkChaosName, "AllInjected")
		switchedNameIsInjected := getConditionStatus(ctx, networkChaosNameSwitched, "AllInjected")

		if (isSelected && isInjected) || (switchedNameIsSelected && switchedNameIsInjected) {
			break
		}

		// Delay between NetworkChaos queries
		time.Sleep(500 * time.Millisecond)
	}
}

func getConditionStatus(ctx context.Context, networkChaosName string, conditionType string) bool {
	cmdArgs := fmt.Sprintf("kubectl describe networkchaos %s | grep -B 1 \"Type:.*%s\" | awk '/Status:/ {print $2}'", networkChaosName, conditionType)
	cmd := exec.CommandContext(ctx, "bash", "-c", cmdArgs)

	cmdOutput, err := cmd.CombinedOutput()
	cmdResult := strings.TrimSpace(string(cmdOutput))
	conditionTrue := cmdResult == "True"
	fmt.Printf("%s: %s\n", conditionType, cmdResult)
	handle(err)

	return conditionTrue
}


