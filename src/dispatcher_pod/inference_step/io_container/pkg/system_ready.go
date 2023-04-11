package dispatcher_inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Dat-Boi-Arjun/SEIFER/io_util/test_util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ReadyJSON struct {
	Ready bool
}

// systemReadinessCheck waits until all inference pods are ready and (if in a test environment) if the ChaosMesh
// bandwidth limits are active, then it writes to a shared file to indicate to the python runtime that it's ready
func systemReadinessCheck(ctx context.Context, dispatcherNode string, dispatcherNextNode string) {

	clientset := getKubeClient()

	podsClient := clientset.CoreV1().Pods(v1.NamespaceDefault)

	inferencePodListOptions := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(
			map[string]string{
				"task": "inference",
				"type": "compute",
			},
		).String(),
	}

	podObjectList, err := podsClient.List(ctx, inferencePodListOptions)
	handle(err)

	podsLeftToCheck := make(map[string]struct{}, len(podObjectList.Items))
	for _, pod := range podObjectList.Items {
		podsLeftToCheck[pod.Name] = struct{}{}
	}

	// Don't specify a resourceVersion - we want all events from the inferencePodsWatcher b/c a pod may have become ready before we
	// made the watch() call
	inferencePodsWatcher, err := podsClient.Watch(ctx, inferencePodListOptions)
	handle(err)

	for event := range inferencePodsWatcher.ResultChan() {
		pod := event.Object.(*v1.Pod)
		fmt.Printf("Event for pod %s\n", pod.Name)
		for _, cond := range pod.Status.Conditions {
			if cond.Type == v1.PodReady && cond.Status == v1.ConditionTrue {
				fmt.Printf("Pod %s became ready \n", pod.Name)
				// If the pod was previously deleted from the podsLeftToCheck list, delete() won't do anything,
				// so it's fine if there are multiple PodReady events for the same pod
				delete(podsLeftToCheck, pod.Name)
			}
		}
		if len(podsLeftToCheck) == 0 {
			inferencePodsWatcher.Stop()
			break
		}
	}

	test_util.WaitForChaosMeshRunning(ctx, dispatcherNode, dispatcherNextNode)
	fmt.Println("ChaosMesh ready")

	// Write ready.txt file to pod volume, so python runtime knows it can start sending inference data
	sendReadinessMessage()
	fmt.Println("Wrote to readiness file")
}

func getKubeClient() *kubernetes.Clientset {
	config, err := rest.InClusterConfig()
	handle(err)
	fmt.Println("Client authenticated in-cluster")

	clientset, err := kubernetes.NewForConfig(config)
	handle(err)

	return clientset
}

func sendReadinessMessage() {
	ready := &ReadyJSON{
		Ready: true,
	}

	jsonData, err := json.Marshal(ready)
	handle(err)

	err = os.WriteFile(readinessFile, jsonData, 0644)
	handle(err)
}
