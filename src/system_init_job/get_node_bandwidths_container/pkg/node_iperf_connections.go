package node_bandwidths

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	sockets "github.com/Dat-Boi-Arjun/SEIFER/io_util"
	"github.com/Dat-Boi-Arjun/SEIFER/io_util/test_util"
	"github.com/pbnjay/memory"
)

const (
	DispatcherHost        string = "dispatcher.default.svc.cluster.local"
	OrchestratorPort      int    = 4000
	ReceiveSystemInfoPort int    = 3000
	ServerType            string = "tcp"
)

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func runServer(ctx context.Context, wg *sync.WaitGroup, numNodes int) {
	// Run the server instance n times, then exit
	for i := 0; i < numNodes; i++ {
		// This server instance will exit after getting a single connection
		// -B 0.0.0.0 binds the server to all incoming network interfaces
		interIP := "0.0.0.0"

		args := fmt.Sprintf("-s -B %s -4 -J --one-off", interIP)
		fmt.Println("Started IPerf server")
		cmd := exec.CommandContext(ctx, "iperf3", strings.Fields(args)...)
		fmt.Printf("Command: %s\n", cmd.String())
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
		}
	}
	wg.Done()
}

func startConnection(ctx context.Context, connectToNode string) float64 {
	// DNS lookup for the service that controls the next inference pod
	IPs, err := net.LookupIP(fmt.Sprintf("node-%s.default.svc.cluster.local", connectToNode))
	// If the other pods haven't been created, the corresponding services won't have DNS records for them
	for err != nil || len(IPs) == 0 {
		fmt.Println(err.Error())
		IPs, err = net.LookupIP(fmt.Sprintf("node-%s.default.svc.cluster.local", connectToNode))
	}
	ip := IPs[0].String()
	timeoutSec := 15
	inter, err := net.InterfaceByName("eth0")
	handle(err)
	addrs, err := inter.Addrs()
	handle(err)
	interIP := addrs[0].(*net.IPNet).IP.String()
	fmt.Printf("eth0 interface: %s\n", interIP)
	// Add -B eth0 to make iperf use the eth0 interface
	args := fmt.Sprintf("-c %s -B %s -4 -J -i 0 -t 10 --connect-timeout=%d", ip, interIP, timeoutSec*1000)
	fmt.Printf("Connecting to %s\n", ip)
	cmd := exec.CommandContext(ctx, "iperf3", strings.Fields(args)...)
	fmt.Printf("Command: %s\n", cmd.String())
	out, err := cmd.Output()
	report := make(map[string]interface{})
	_ = json.Unmarshal(out, &report)
	for err != nil || (report["error"] != nil && report["error"].(string) != "") {
		fmt.Println("Retrying connection")
		cmd = exec.CommandContext(ctx, "iperf3", strings.Fields(args)...)
		out, err = cmd.Output()
		report = make(map[string]interface{})
		_ = json.Unmarshal(out, &report)
	}
	fmt.Println("Got report")
	fmt.Println(string(out))
	bandwidthSent := report["end"].(map[string]interface{})["sum_sent"].(map[string]interface{})["bits_per_second"].(float64)
	bandwidthRecv := report["end"].(map[string]interface{})["sum_received"].(map[string]interface{})["bits_per_second"].(float64)
	avgBandwidth := (bandwidthSent + bandwidthRecv) / 2
	fmt.Printf("Bandwidth: %f\n", avgBandwidth)
	fmt.Printf("Finished %s\n", connectToNode)
	return avgBandwidth
}

func RunBandwidthTasks(ctx context.Context) {
	NodeName := os.Getenv("NODE_NAME")
	numSystemNodes, err := strconv.Atoi(os.Getenv("NUM_NODES"))
	handle(err)

	var otherNodes []string
	err = json.Unmarshal([]byte(os.Getenv("OTHER_NODES")), &otherNodes)
	handle(err)
	fmt.Printf("Other nodes: %s\n", strings.Join(otherNodes, ","))

	var wg sync.WaitGroup
	wg.Add(1)
	// The first node will have no other nodes connecting to it,
	// the second node will have 1 other node connecting to it, etc.
	numServerConnections := numSystemNodes - (len(otherNodes) + 1)
	go runServer(ctx, &wg, numServerConnections)
	bandwidthMap := make(map[string]float64)
	// Only fill the bandwidth map if the node actually needs to connect to other nodes
	if len(otherNodes) > 0 {
		bandwidthMap = orchestrateIPerfTasks(ctx, NodeName, otherNodes)
	}

	fmt.Println("Parsing data")
	bandwidthInfo := make([]*BandwidthInfo, 0, len(bandwidthMap))
	for otherNode, bandwidth := range bandwidthMap {
		info := &BandwidthInfo{
			Start:     NodeName,
			End:       otherNode,
			Bandwidth: bandwidth,
		}
		bandwidthInfo = append(bandwidthInfo, info)
	}

	mem := memory.TotalMemory()

	nodesInfo := &NodeInfo{
		Bandwidths: bandwidthInfo,
		NodeMemory: mem,
	}

	jsonArray, err := json.Marshal(nodesInfo)
	handle(err)
	fmt.Println("Parsed data")

	fmt.Println("Sending data to dispatcher")
	sendToDispatcher(jsonArray)
	fmt.Println("Sent data to dispatcher")

	fmt.Println("Keeping server alive")
	wg.Wait()
}

func orchestrateIPerfTasks(ctx context.Context, nodeName string, otherNodes []string) map[string]float64 {
	// The key is the node, the value is the bandwidth
	bandwidthMap := make(map[string]float64)
	fmt.Println("Connecting to orchestrator")
	// Dial to the orchestrator on the dispatcher server
	connection, err := net.Dial(ServerType, net.JoinHostPort(DispatcherHost, strconv.Itoa(OrchestratorPort)))
	var dnsError *net.DNSError
	for errors.Is(err, syscall.ECONNREFUSED) || errors.As(err, &dnsError) {
		fmt.Println("Connection refused, retrying")
		connection, err = net.Dial(ServerType, net.JoinHostPort(DispatcherHost, strconv.Itoa(OrchestratorPort)))
	}
	handle(err)
	var wr io.WriteCloser = connection
	fmt.Println("Writing node name")
	err = sockets.WriteOutput(&wr, []byte(nodeName))
	handle(err)
	fmt.Println("Connected to orchestrator")
	var reader io.ReadCloser = connection
	for range otherNodes {
		inpt, _ := sockets.ReadInput(&reader)
		connectTo := string(inpt)
		fmt.Printf("Connecting to %s\n", connectTo)
		fmt.Println("Waiting for chaos mesh to activate")
		test_util.WaitForChaosMeshRunning(ctx, nodeName, connectTo)
		// Wait 0.5 sec extra to let the TBF rules take effect, just in case
		time.Sleep(500 * time.Millisecond)
		fmt.Println("ChaosMesh activated")
		bandwidth := startConnection(ctx, connectTo)
		fmt.Printf("Finished job on %s\n", connectTo)
		bandwidthMap[connectTo] = bandwidth
		// Acknowledge that we ran the iperf job
		var ack uint8 = 6
		n := 0
		for n < 1 {
			n, _ = connection.Write([]byte{ack})
		}
		fmt.Println("Informed orchestrator that job finished")
	}
	return bandwidthMap
}

func sendToDispatcher(jsonData []byte) {
	connection, err := net.Dial(ServerType, net.JoinHostPort(DispatcherHost, strconv.Itoa(ReceiveSystemInfoPort)))
	var dnsError *net.DNSError
	for errors.Is(err, syscall.ECONNREFUSED) || errors.As(err, &dnsError) {
		fmt.Println("Connection refused, retrying")
		connection, err = net.DialTimeout(ServerType, net.JoinHostPort(DispatcherHost, strconv.Itoa(ReceiveSystemInfoPort)), 10*time.Second)
	}
	handle(err)

	var writer io.WriteCloser = connection
	err = sockets.WriteOutput(&writer, jsonData)
	handle(err)
}
