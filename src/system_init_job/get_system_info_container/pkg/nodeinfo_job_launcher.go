package system_info

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
)

const (
	ReceiveSystemInfoPort int = 3000
)

func createJob(ctx context.Context, clientset *kubernetes.Clientset, nodes []string, nodeName string) {
	otherNodes := make([]string, 0, len(nodes)-1)
	for _, node := range nodes {
		if node != nodeName {
			otherNodes = append(otherNodes, node)
		}
	}
	otherNodesJson, _ := json.Marshal(otherNodes)

	jobClient := clientset.BatchV1().Jobs(corev1.NamespaceDefault)
	var terminateAfter int32 = 0

	jobName := fmt.Sprintf("%s-bandwidths", nodeName)
	jobSpec := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: "default",
			Labels: map[string]string{
				"task":          "system-info",
				"assigned-node": nodeName,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: "default",
					Labels: map[string]string{
						"task":          "system-info",
						"assigned-node": nodeName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  jobName,
							Image: "ghcr.io/dat-boi-arjun/get_node_bandwidths:latest",
							Env: []corev1.EnvVar{
								{
									Name:  "OTHER_NODES",
									Value: string(otherNodesJson),
								},
								{
									Name:  "NODE_NAME",
									Value: nodeName,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: int32(5201),
									Protocol:      "TCP",
								},
							},
						},
					},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: "ghcr-imagepull-auth",
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
					NodeName:      nodeName,
				},
			},
			TTLSecondsAfterFinished: &terminateAfter,
		},
	}

	_, err := jobClient.Create(ctx, jobSpec, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		deletePods := metav1.DeletionPropagation("Background")
		err = jobClient.Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &deletePods})
		handle(err)
		_, err = jobClient.Create(ctx, jobSpec, metav1.CreateOptions{})
		handle(err)
	} else {
		handle(err)
	}
}

func createDispatcherService(ctx context.Context, clientset *kubernetes.Clientset) {
	serviceClient := clientset.CoreV1().Services(corev1.NamespaceDefault)

	serviceSpec := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "dispatcher",
			// So we can delete services
			Labels: map[string]string{
				"app": "defer",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				// Match w/ all pods that are doing some dispatcher-related functionality
				// System init, inference pod deployment, dispatching inference data
				"type": "dispatcher",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "system-info-input",
					Protocol: corev1.Protocol("TCP"),
					Port:     int32(ReceiveSystemInfoPort),
				},
				{
					Name:     "iperf-orchestrator",
					Protocol: corev1.Protocol("TCP"),
					Port:     int32(OrchestratorPort),
				},
				{
					Name:     "finished-inference",
					Protocol: corev1.Protocol("TCP"),
					// Standard port for inference pod
					Port: 8080,
				},
				// Figure out nodePort when inferencing works
				/*{
					Name:     "external-input",
					Protocol: corev1.Protocol("TCP"),
					Port:     32000,
					// Exposed externally, need to print the node IP of the service
					NodePort: 32000,
				},*/
			},
		},
	}

	_, err := serviceClient.Create(ctx, serviceSpec, metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		err := serviceClient.Delete(ctx, "dispatcher", metav1.DeleteOptions{})
		handle(err)
		_, err = serviceClient.Create(ctx, serviceSpec, metav1.CreateOptions{})
		handle(err)
	}
	fmt.Println("Dispatcher Service created")
}

func createServices(ctx context.Context, clientset *kubernetes.Clientset, nodes []string) {
	fmt.Println("Creating dispatcher service")
	createDispatcherService(ctx, clientset)

	serviceClient := clientset.CoreV1().Services(corev1.NamespaceDefault)

	for _, node := range nodes {
		serviceName := fmt.Sprintf("node-%s", node)
		// Each partition gets its own service to be able to talk to other pods
		partitionService := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name: serviceName,
				Labels: map[string]string{
					"app": "defer",
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"assigned-node": node,
				},
				Ports: []corev1.ServicePort{
					{
						Name:     "inference-input",
						Protocol: "TCP",
						// Standard port used across inference pods
						Port: 8080,
					},
					{
						Name:     "iperf-server-port",
						Protocol: "TCP",
						// Standard port used across inference pods
						Port: 5201,
					},
				},
			},
		}

		_, err := serviceClient.Create(ctx, partitionService, metav1.CreateOptions{})
		if err != nil && errors.IsAlreadyExists(err) {
			err = serviceClient.Delete(ctx, serviceName, metav1.DeleteOptions{})
			handle(err)
			_, err = serviceClient.Create(ctx, partitionService, metav1.CreateOptions{})
			handle(err)
		} else {
			handle(err)
		}
		fmt.Printf("Created %s service\n", serviceName)
	}
}

func LaunchJobs(ctx context.Context, clientset *kubernetes.Clientset, nodes []string) {
	fmt.Println("LaunchJobs() running")

	// Create the dispatcher and each compute node's service
	createServices(ctx, clientset, nodes)

	fmt.Println("Services created, now creating jobs")
	for _, node := range nodes {
		createJob(ctx, clientset, nodes, node)
		fmt.Printf("Created job for node %s\n", node)
	}
	fmt.Println("LaunchJobs() finished")
}
