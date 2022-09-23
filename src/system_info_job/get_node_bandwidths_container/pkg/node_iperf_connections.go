package node_bandwidths

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	sockets "github.com/Dat-Boi-Arjun/DEFER/io_util"
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
		args := "-s -4 -J --one-off"
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
	IPs, _ := net.LookupIP(fmt.Sprintf("node-%s.default.svc.cluster.local", connectToNode))
	ip := IPs[0].String()
	timeoutSec := 15
	args := fmt.Sprintf("-c %s -4 -J -i 0 -t 10 --connect-timeout=%d", ip, timeoutSec*1000)
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

	var otherNodes []string
	err := json.Unmarshal([]byte(os.Getenv("OTHER_NODES")), &otherNodes)
	handle(err)
	fmt.Printf("Other nodes: %s\n", strings.Join(otherNodes, ","))
	numNodes := len(otherNodes)

	var wg sync.WaitGroup
	wg.Add(1)
	go runServer(ctx, &wg, numNodes)
	bandwidthMap := orchestrateIPerfJobs(ctx, NodeName, otherNodes)

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

func orchestrateIPerfJobs(ctx context.Context, nodeName string, otherNodes []string) map[string]float64 {
	// The key is the node, the value is the bandwidth
	bandwidthMap := make(map[string]float64)
	fmt.Println("Connecting to orchestrator")
	// Dial to the orchestrator on the dispatcher server
	connection, err := net.Dial(ServerType, net.JoinHostPort(DispatcherHost, strconv.Itoa(OrchestratorPort)))
	handle(err)
	var wr io.WriteCloser = connection
	fmt.Println("Writing node name")
	sockets.WriteOutput(&wr, []byte(nodeName))
	fmt.Println("Connected to orchestrator")
	var reader io.ReadCloser = connection
	for range otherNodes {
		inpt, _ := sockets.ReadInput(&reader)
		connectTo := string(inpt)
		fmt.Printf("Connecting to %s\n", connectTo)
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
	handle(err)

	var writer io.WriteCloser = connection
	sockets.WriteOutput(&writer, jsonData)
}
