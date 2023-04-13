package inference_io

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	pipesutil "github.com/Dat-Boi-Arjun/SEIFER/io_util/pipes_util"
	sockets "github.com/Dat-Boi-Arjun/SEIFER/io_util/sockets_util"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var node = os.Getenv("NODE")
var nextNode = os.Getenv("NEXT_NODE")

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func RunSockets() {
	fmt.Println("Started running inference io")
	// Let all processes know when we exit, so we can close all pipes/sockets
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	config, err := rest.InClusterConfig()
	handle(err)
	clientset, err := kubernetes.NewForConfig(config)
	handle(err)
	fmt.Println("Authenticated in-cluster")

	c := make(chan os.Signal)
	// The program won't exit on its own
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var sendPipe io.WriteCloser
	var recvPipe io.ReadCloser
	var readSock io.ReadCloser
	var writeSock io.WriteCloser

	// Connect to inference runtime first, then sockets second
	pipePath := "/io"
	sendPath := filepath.Join(pipePath, "/to_inference")
	recvPath := filepath.Join(pipePath, "/from_inference")

	fmt.Println("Creating pipes")
	sendPipe = pipesutil.CreatePipe(sendPath, "send")
	recvPipe = pipesutil.CreatePipe(recvPath, "recv")

	toInferenceTransferInfo := sockets.TransferDetails{
		From: sockets.TransferType{
			Medium: "socket",
			// No connect info to fill
		},
		To: sockets.TransferType{
			Medium:      "pipe",
			ConnectInfo: sendPath,
		},
	}

	// Create all connections concurrently
	fmt.Println("Connecting to sockets")
	cCli, cServ := make(chan *net.Conn, 1), make(chan *net.Conn, 1)

	go sockets.CreateClientSocket(cCli, nextNode)
	go sockets.CreateServerSocket(cServ)

	fmt.Println("Waiting on connections from channel")
	for i := 0; i < 2; i++ {
		select {
		case data := <-cCli:
			writeSock = *data
		case data := <-cServ:
			readSock = *data
		}
	}
	fmt.Println("Got connections from channel")

	inferencePodReadinessCheck(ctx, clientset, node, nextNode)

	fmt.Println("Launching transfer to inference")
	// Incoming socket data -> inference
	go sockets.Transfer(ctx, &readSock, &sendPipe, toInferenceTransferInfo)

	fromInferenceTransferInfo := sockets.TransferDetails{
		From: sockets.TransferType{
			Medium:      "pipe",
			ConnectInfo: recvPath,
		},
		To: sockets.TransferType{
			Medium:      "socket",
			ConnectInfo: nextNode,
		},
	}
	fmt.Println("Launching transfer from inference")
	// Outgoing inference data -> next node
	go sockets.Transfer(ctx, &recvPipe, &writeSock, fromInferenceTransferInfo)

	fmt.Println("Waiting for context to finish")
	select {
	case <-c:
		// Cancel the context and all the functions
		cancel()
	}
}
