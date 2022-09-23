package pipes_util

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func handle(e error) {
	if e != nil {
		panic(e)
	}
}

func CreatePipe(path string, mode string) *os.File {
	// Golang opens it in non-blocking mode (to prevent deadlock on opening)
	var flags int
	if mode == "send" {
		// The writer should be in blocking mode so that it waits for the reader to open the pipe
		flags = syscall.O_CREAT | syscall.O_WRONLY
	}
	if mode == "recv" {
		// The reader needs to open the pipe and be in non-blocking mode
		err := syscall.Mkfifo(path, 0666)
		if err != nil && !errors.Is(err, os.ErrExist) {
			panic(err)
		}
		flags = syscall.O_NONBLOCK | syscall.O_CREAT | syscall.O_RDONLY
	}

	fmt.Println("Trying to open pipe")
	pipe, err := os.OpenFile(path, flags, os.ModeNamedPipe)
	for os.IsNotExist(err) {
		pipe, err = os.OpenFile(path, flags, os.ModeNamedPipe)
	}
	handle(err)

	fmt.Println("Opened pipe")
	return pipe
}
