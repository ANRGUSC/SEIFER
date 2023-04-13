package inference_io

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Dat-Boi-Arjun/SEIFER/io_util/test_util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// inferencePodReadinessCheck is called after the inference sockets are set up, so the only thing it checks is whether the
// ChaosMesh bandwidth limit is active
func inferencePodReadinessCheck(ctx context.Context, clientset *kubernetes.Clientset, node string, otherNode string) {
	fmt.Println("Waiting for ChaosMesh to be running")
	var nextNode string
	if otherNode == "dispatcher" {
		nextNode = os.Getenv("DISPATCHER_NAME")
	} else {
		nextNode = otherNode
	}

	test_util.WaitForChaosMeshRunning(ctx, node, nextNode)
	fmt.Println("ChaosMesh successfully running")

	sendReadinessMessage(ctx, clientset)
	fmt.Println("Updated pod status indicating readiness")
}

func sendReadinessMessage(ctx context.Context, clientset *kubernetes.Clientset) {

	podsClient := clientset.CoreV1().Pods(v1.NamespaceDefault)

	podListOptions := metav1.ListOptions{
		LabelSelector: labels.Set(
			map[string]string{
				"task":          "inference",
				"type":          "compute",
				"assigned-node": node,
			},
		).String(),
	}

	pods, err := podsClient.List(ctx, podListOptions)
	handle(err)
	podName := pods.Items[0].Name

	// Add new PodStatus condition indicating that it's ready for inferencing
	patchInfo := []Cond{{
		Op:   "add",
		Path: "/status/conditions/-",
		Value: Value{
			Type:   "InferenceReady",
			Status: "True",
		},
	}}

	patchData, _ := json.Marshal(patchInfo)

	_, err = podsClient.Patch(ctx, podName, types.JSONPatchType, patchData, metav1.PatchOptions{}, "status")
	handle(err)
}

type Value struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type Cond struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value Value  `json:"value"`
}
