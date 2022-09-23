package io_util

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"syscall"
)

const intSizeBytes int = 4 // Python might use int64?

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

// Named pipes are non-blocking, sockets are blocking
// handleOtherErrs returns true if it's a blocking error, false if there's no error, and panics for other errors
func handleOtherErrs(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) {
		return true
	} else {
		panic(err)
	}
	return false
}

// ReadInput Read data from IO, abstracted to work for both FIFO and sockets
func ReadInput(reader *io.ReadCloser) ([]byte, error) {
	dataSizeArr := make([]byte, intSizeBytes)
	sizeLeftToRead := intSizeBytes
	// Looping until we read all 4 bytes of the data size
	for sizeLeftToRead > 0 {
		amtRead, err := (*reader).Read(dataSizeArr[intSizeBytes-sizeLeftToRead:])
		if errors.Is(err, syscall.ECONNRESET) {
			return nil, err
		}
		// If there wasn't data to read, try again on next loop
		if handleOtherErrs(err) == true {
			continue
		}
		if amtRead > 0 {
			fmt.Printf("Size read: %d\n", amtRead)
		}
		sizeLeftToRead -= amtRead
	}
	var dataSize int32
	fmt.Println(dataSizeArr)
	buf := bytes.NewReader(dataSizeArr)
	err := binary.Read(buf, binary.BigEndian, &dataSize)
	handle(err)
	fmt.Printf("Data to read: %d\n", dataSize)

	data := make([]byte, dataSize)
	dataLeftToRead := dataSize
	for dataLeftToRead > 0 {
		amtRead, err := (*reader).Read(data[dataSize-dataLeftToRead:])
		if errors.Is(err, syscall.ECONNRESET) {
			return nil, err
		}
		// If there wasn't data to read, try again on next loop
		if handleOtherErrs(err) == true {
			continue
		}
		fmt.Printf("Data read: %d\n", amtRead)
		dataLeftToRead -= int32(amtRead)
	}

	return data, nil
}

// WriteOutput Write data to IO, abstracted to work for both FIFO and sockets
func WriteOutput(writer *io.WriteCloser, dataIn []byte) error {
	data := make([]byte, len(dataIn))
	copy(data, dataIn)
	dataSize := len(data)
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, int32(dataSize))
	handle(err)
	sizeBytes := make([]byte, buf.Len())
	copy(sizeBytes, buf.Bytes())
	fmt.Println(sizeBytes)
	sizeLeftToWrite := intSizeBytes
	for sizeLeftToWrite > 0 {
		amtWr, err := (*writer).Write(sizeBytes[intSizeBytes-sizeLeftToWrite:])
		if errors.Is(err, syscall.EPIPE) {
			return err
		}
		// If couldn't write data, try again on next loop
		if handleOtherErrs(err) == true {
			continue
		}
		fmt.Printf("Size written: %d\n", amtWr)
		sizeLeftToWrite -= amtWr
	}
	fmt.Printf("Data to write: %d\n", dataSize)
	dataLeftToWrite := dataSize
	for dataLeftToWrite > 0 {
		//fmt.Printf("Data left to write: %d\n", dataLeftToWrite)
		amtWr, err := (*writer).Write(data[dataSize-dataLeftToWrite:])
		if errors.Is(err, syscall.EPIPE) {
			return err
		}
		// If couldn't write data, try again on next loop
		if handleOtherErrs(err) == true {
			continue
		}
		if amtWr > 0 {
			fmt.Printf("Data written: %d\n", amtWr)
		}
		dataLeftToWrite -= amtWr
	}
	return nil
}
