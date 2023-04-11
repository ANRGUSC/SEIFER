package system_info

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"github.com/Dat-Boi-Arjun/SEIFER/io_util"
)

const (
	OrchestratorPort = 4000
)

func GetConnectionsToNode(nodes []string) map[string][]string {
	numNodes := len(nodes)
	connectionsToNode := make(map[string][]string)
	// Every node except the last node has connections
	for i, node := range nodes[:numNodes-1] {
		// The max connections to a node will be the first node, which will have n-1 connections
		// The second node will have n-2 connections and so on...
		connectionsToNode[node] = make([]string, 0, (numNodes-1)-i)
		// Node i will be the current node, so nodes[i+1:] has the rest of the nodes in the list
		for _, otherNode := range nodes[i+1:] {
			connectionsToNode[node] = append(connectionsToNode[node], otherNode)
		}
	}

	// Last node is part of the map but has no connections
	lastNode := nodes[numNodes-1]
	connectionsToNode[lastNode] = make([]string, 0)

	return connectionsToNode
}

func Run(mainWg *sync.WaitGroup, nodes []string, connectionsToNode map[string][]string) {
	fmt.Println("Orchestrating connections")
	server, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(OrchestratorPort)))
	handle(err)
	numNodes := len(nodes)

	nodeConns := make(map[string]*net.Conn)

	fmt.Println("Waiting for orchestrator connections")
	// Every node except the last node needs to connect and run bandwidth jobs
	for len(nodeConns) < numNodes-1 {
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
	// Every node except the last needs to orchestrate IPerf connections
	for _, node := range nodes[:len(nodes)-1] {
		fmt.Printf("Orchestrating connections for %s\n", node)
		for _, otherNode := range connectionToNodes[node] {
			// Tell node it can connect to the iperf server on otherNode
			conn := *nodeConns[node]
			var writer io.WriteCloser = conn
			fmt.Printf("Telling %s to run job on %s\n", node, otherNode)
			err := io_util.WriteOutput(&writer, []byte(otherNode))
			handle(err)
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
