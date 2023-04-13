package main

import (
	"context"
	"fmt"
	"os/exec"

	deploypods "github.com/Dat-Boi-Arjun/SEIFER/dispatcher_pod/config_step/deploy_container/pkg"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(context.Background())
	config, err := rest.InClusterConfig()
	handle(err)
	fmt.Println("Main authenticated in-cluster")

	clientset, err := kubernetes.NewForConfig(config)
	handle(err)

	// Create NetworkChaos for inference step
	cmd := exec.Command("/bin/sh", "/root/cluster_test.sh")
	out, err := cmd.CombinedOutput()
	fmt.Println(string(out))

	deploypods.DeployDispatcherInferencePod(ctx, clientset)
	deploypods.DeployInferencePods(ctx, clientset)

	// Stop any residual processes
	cancel()
}
