package system_info

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"path/filepath"
	"strconv"
	"sync"

	sockets "github.com/Dat-Boi-Arjun/SEIFER/io_util"
	systeminfo "github.com/Dat-Boi-Arjun/SEIFER/system_init_job/get_node_bandwidths_container/pkg"
)

const (
	// ServerPort Receive system info data on port 3000
	ServerPort int    = 3000
	ServerType string = "tcp"
)

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func ReceiveData(wg *sync.WaitGroup, NumNodes int) {
	fmt.Println("ReceiveData() running")

	bandwidthData := make([]*systeminfo.BandwidthInfo, 0)
	server, err := net.Listen(ServerType, net.JoinHostPort("", strconv.Itoa(ServerPort)))
	handle(err)
	fmt.Println("Started dispatcher server to receive node info")

	// Find the smallest available memory across all nodes, this will become the available memory for each node
	memChan := make(chan uint64, NumNodes)
	bandwidthChan := make(chan []*systeminfo.BandwidthInfo, NumNodes)
	var availableMem uint64 = math.MaxUint64
	fmt.Printf("Listening on %s\n", server.Addr().String())
	var recvWg sync.WaitGroup
	recvWg.Add(NumNodes)
	for i := 0; i < NumNodes; i++ {
		fmt.Printf("Receiver waiting for connection %d\n", i+1)
		connection, err := server.Accept()
		fmt.Println("Receiver accepted connection from node")
		handle(err)
		go handleConnection(&connection, &recvWg, bandwidthChan, memChan)
	}
	// Wait for all nodes to connect and send their bandwidth data
	recvWg.Wait()

	for len(memChan) > 0 || len(bandwidthChan) > 0 {
		select {
		case mem := <-memChan:
			if mem < availableMem {
				availableMem = mem
			}
		case bandwidths := <-bandwidthChan:
			bandwidthData = append(bandwidthData, bandwidths...)
		}
	}

	err = server.Close()
	handle(err)

	nodesData := map[string]interface{}{
		"bandwidths":    bandwidthData,
		"node_capacity": availableMem,
		"num_nodes":     NumNodes,
	}

	fileJson, err := json.Marshal(nodesData)
	handle(err)
	dir := "/config"
	err = ioutil.WriteFile(filepath.Join(dir, "node_info.json"), fileJson, 0644)
	handle(err)

	wg.Done()
}

func handleConnection(conn *net.Conn, wg *sync.WaitGroup, bandwidthChan chan []*systeminfo.BandwidthInfo, memChan chan uint64) {
	var reader io.ReadCloser = *conn
	input, _ := sockets.ReadInput(&reader)
	var nodeInfo systeminfo.NodeInfo
	err := json.Unmarshal(input, &nodeInfo)
	handle(err)

	bandwidthChan <- nodeInfo.Bandwidths
	memChan <- nodeInfo.NodeMemory

	err = (*conn).Close()
	handle(err)
	wg.Done()
	fmt.Println("Dispatcher received info and handled connection")
}
