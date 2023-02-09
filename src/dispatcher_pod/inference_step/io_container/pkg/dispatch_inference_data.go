package dispatcher_inference

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	pipesutil "github.com/Dat-Boi-Arjun/DEFER/io_util/pipes_util"
	socketsutil "github.com/Dat-Boi-Arjun/DEFER/io_util/sockets_util"
)

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func RunSockets() {
	var firstNode, _ = ioutil.ReadFile("/nfs/dispatcher_config/dispatcher_next_node.txt")
	sendTo := string(firstNode)

	// Let all processes know when we exit, so we can stop all pods
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)

	// TODO figure out how to let program know if we want to deploy a new model
	// Maybe create container to watch for updates to the model server?
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	var inferenceReader io.ReadCloser
	var writeSock io.WriteCloser
	var readPipe io.ReadCloser
	var writePipe io.WriteCloser

	// Connect to processing runtime first before sockets
	path := "/io"
	sendPath := filepath.Join(path, "/to_processing")
	recvPath := filepath.Join(path, "/from_processing")
	// Connection to Python preprocessing container
	fmt.Println("Creating pipes")
	writePipe = pipesutil.CreatePipe(sendPath, "send")
	readPipe = pipesutil.CreatePipe(recvPath, "recv")

	fromProcessingTransferInfo := socketsutil.TransferDetails{
		From: socketsutil.TransferType{
			Medium:      "pipe",
			ConnectInfo: recvPath,
		},
		To: socketsutil.TransferType{
			Medium:      "socket",
			ConnectInfo: sendTo,
		},
	}

	// Needs to be concurrent so other nodes can connect to the server too
	cCli, cServ := make(chan *net.Conn, 1), make(chan *net.Conn, 1)
	// Create all connections concurrently
	go socketsutil.CreateClientSocket(cCli, sendTo)

	// Connection from the last compute node
	go socketsutil.CreateServerSocket(cServ)

	for i := 0; i < 2; i++ {
		select {
		case data := <-cCli:
			writeSock = *data
		case data := <-cServ:
			inferenceReader = *data
		}
	}

	fmt.Println("Launching transfer to first compute node")
	// Incoming data -> first compute node
	go socketsutil.Transfer(ctx, &readPipe, &writeSock, fromProcessingTransferInfo)

	toProcessingTransferInfo := socketsutil.TransferDetails{
		From: socketsutil.TransferType{
			Medium: "socket",
		},
		To: socketsutil.TransferType{
			Medium:      "pipe",
			ConnectInfo: sendPath,
		},
	}
	fmt.Println("Launching recv finished inference")
	// Model inference -> Python processing
	// User needs to encode the data w/ ZFP and use LZ4 compression
	go socketsutil.Transfer(ctx, &inferenceReader, &writePipe, toProcessingTransferInfo)

	select {
	case <-c:
		// Cancel the context and all the functions
		cancel()
	}
}
