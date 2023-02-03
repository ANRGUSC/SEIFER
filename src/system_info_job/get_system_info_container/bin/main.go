package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Dat-Boi-Arjun/DEFER/system_info_job/get_system_info_container/pkg"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	fmt.Println("Running get_system_info main")
	var dispatcherName = os.Getenv("DISPATCHER_NAME")
	ctx, cancel := context.WithCancel(context.Background())
	config, err := rest.InClusterConfig()
	handle(err)
	fmt.Println("Main authenticated in-cluster")

	clientset, err := kubernetes.NewForConfig(config)
	handle(err)

	list, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	handle(err)

	// Number of compute nodes
	NumNodes := len(list.Items) - 1
	fmt.Printf("Num compute nodes: %d\n", NumNodes)

	otherNodes := make([]string, 0, NumNodes)
	for _, item := range list.Items {
		if item.Name != dispatcherName {
			otherNodes = append(otherNodes, item.Name)
		}
	}

	// Use existing node bandwidth information
	/*system_info.LaunchJobs(ctx, clientset, otherNodes)
	var wg sync.WaitGroup
	wg.Add(2)
	go system_info.Run(ctx, &wg, clientset, otherNodes)
	go system_info.ReceiveData(&wg, NumNodes)
	wg.Wait()*/

	system_info.InitDispatcher(ctx, clientset, dispatcherName)

	// Stop any residual processes
	cancel()
}
