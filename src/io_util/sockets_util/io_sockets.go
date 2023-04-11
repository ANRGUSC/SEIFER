package sockets_util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"syscall"
	"time"

	"github.com/Dat-Boi-Arjun/SEIFER/io_util"
	"github.com/Dat-Boi-Arjun/SEIFER/io_util/pipes_util"
)

// Standardized across all inference io
const (
	ServerPort int    = 8080
	ServerType string = "tcp"
)

// TransferDetails The transfer details of the two mediums we're transferring between
// Can be passed by value since it won't be modified inside Transfer()
type TransferDetails struct {
	From TransferType
	To   TransferType
}

// Defines the transfer type
type TransferType struct {
	// "socket" or "pipe"
	Medium string
	// If pipe, the filepath of the pipe, if socket, the hostname of the connection
	ConnectInfo string
}

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func Transfer(ctx context.Context, inputReader *io.ReadCloser, outputWriter *io.WriteCloser, transferInfo TransferDetails) {
	reader := inputReader
	writer := outputWriter
	// buffer size of 10^5
	c := make(chan []byte, int(math.Pow10(5)))
	go func() {
		for {
			select {
			case <-ctx.Done():
				err := (*reader).Close()
				handle(err)
				close(c)
				break
			// If the context isn't closed, we send data like normal
			default:
				input, err := io_util.ReadInput(reader)
				// Handling connection reset error
				if err != nil {
					switch transferInfo.From.Medium {
					case "socket":
						fmt.Println("Recv socket error")
						cServ := make(chan *net.Conn, 1)
						CreateServerSocket(cServ)
						var conn io.ReadCloser = *<-cServ
						reader = &conn
					case "pipe":
						fmt.Println("Recv pipe error")
						pipes_util.CreatePipe(transferInfo.From.ConnectInfo, "recv")
					}
				} else {
					fmt.Printf("Got input from %s, sending thru channel\n", transferInfo.From.Medium)
					c <- input
					fmt.Printf("Channel len after send: %d\n", len(c))
				}
			}
		}
	}()

	go func() {
		for {
			fmt.Printf("Channel len: %d\n", len(c))
			select {
			case <-ctx.Done():
				fmt.Println("Context done")
				err := (*writer).Close()
				handle(err)
				break
			// If the context isn't done, we just receive data like normal
			case data := <-c:
				fmt.Println("Got data from channel, writing")
				err := io_util.WriteOutput(writer, data)
				// Handling broken pipe error
				if err != nil {
					switch transferInfo.To.Medium {
					case "socket":
						fmt.Println("Client socket error")
						cCli := make(chan *net.Conn, 1)
						CreateClientSocket(cCli, transferInfo.To.ConnectInfo)
						var conn io.WriteCloser = *<-cCli
						writer = &conn
					case "pipe":
						fmt.Println("Send pipe error")
						pipes_util.CreatePipe(transferInfo.To.ConnectInfo, "send")
					}
				} else {
					fmt.Printf("Sent output to %s\n", transferInfo.To.Medium)
				}
				break
			}
		}
	}()
}

func CreateClientSocket(c chan *net.Conn, nextNode string) {
	hostname := ""
	// DNS lookup for the service that controls the next inference pod
	if nextNode == "dispatcher" {
		hostname = fmt.Sprintf("%s.default.svc.cluster.local", nextNode)
	} else {
		hostname = fmt.Sprintf("node-%s.default.svc.cluster.local", nextNode)
	}
	connection, err := net.Dial(ServerType, net.JoinHostPort(hostname, strconv.Itoa(ServerPort)))
	// Keep trying connection until we can get through
	var dnsError *net.DNSError
	for errors.Is(err, syscall.ECONNREFUSED) || errors.As(err, &dnsError) {
		fmt.Println("Connection refused, retrying")
		connection, err = net.DialTimeout(ServerType, net.JoinHostPort(hostname, strconv.Itoa(ServerPort)), 10*time.Second)
	}
	handle(err)
	fmt.Println("connected to server")
	c <- &connection
}

func CreateServerSocket(c chan *net.Conn) {
	fmt.Println("Server Running...")
	server, err := net.Listen(ServerType, net.JoinHostPort("", strconv.Itoa(ServerPort)))
	handle(err)
	fmt.Println("Listening on " + server.Addr().String())
	connection, err := server.Accept()
	// After getting the single client connection we can close the listener
	err = server.Close()
	handle(err)
	fmt.Println("client connected")
	c <- &connection
}
