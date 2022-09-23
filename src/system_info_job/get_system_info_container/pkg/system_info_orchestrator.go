package system_info

import (
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/Dat-Boi-Arjun/DEFER/io_util"
	"k8s.io/client-go/kubernetes"
)

const (
	OrchestratorPort = 4000
)

func Run(ctx context.Context, mainWg *sync.WaitGroup, clientset *kubernetes.Clientset, nodes []string) {
	fmt.Println("Orchestrating connections")
	server, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(OrchestratorPort)))
	handle(err)
	numNodes := len(nodes)

	connectionsToNode := make(map[string][]string)
	for _, node := range nodes {
		connectionsToNode[node] = make([]string, 0, numNodes-1)
		for _, otherNode := range nodes {
			if otherNode != node {
				connectionsToNode[node] = append(connectionsToNode[node], otherNode)
			}
		}
	}

	fmt.Println(nodes)

	nodeConns := make(map[string]*net.Conn)

	fmt.Println("Waiting for orchestrator connections")
	for len(nodeConns) < numNodes {
		conn, err := server.Accept()
		fmt.Println(len(nodeConns))
		handle(err)
		ip, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		handle(err)
		fmt.Printf("Got connection to orchestrator: %s\n", ip)
		var r io.ReadCloser = conn
		connected, err := io_util.ReadInput(&r)
		node := string(connected)
		fmt.Printf("Node connected: %s\n", node)
		nodeConns[node] = &conn
	}

	fmt.Println("Orchestrating IPerf connections")
	orchestrateIperfConns(nodes, connectionsToNode, nodeConns)

	mainWg.Done()
}

// orchestrateIperfConns will preside over the iperf server on a certain node and tell other nodes when they can connect
func orchestrateIperfConns(nodes []string, connectionToNodes map[string][]string, nodeConns map[string]*net.Conn) {
	for _, node := range nodes {
		fmt.Printf("Orchestrating connections for %s\n", node)
		for _, otherNode := range connectionToNodes[node] {
			// Tell node it can connect to the iperf server on otherNode
			conn := *nodeConns[node]
			var writer io.WriteCloser = conn
			fmt.Printf("Telling %s to run job on %s\n", node, otherNode)
			io_util.WriteOutput(&writer, []byte(otherNode))
			// Receive acknowledgment of iperf test completion
			read := make([]byte, 1)
			var reader io.Reader = conn
			n := 0
			for n < 1 {
				n, _ = io.ReadFull(reader, read)
			}
			fmt.Printf("%s finished job on %s\n", node, otherNode)
		}
	}
}
