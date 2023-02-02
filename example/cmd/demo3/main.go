package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var sig os.Signal
	ch := make(chan os.Signal, 1)
	signal.Notify(
		ch,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGILL,
		syscall.SIGTRAP,
		syscall.SIGABRT,
		syscall.SIGBUS,
		syscall.SIGFPE,
		// syscall.SIGKILL,
		syscall.SIGSEGV,
		syscall.SIGPIPE,
		syscall.SIGALRM,
		syscall.SIGTERM,
	)
	count := 0
	for {
		select {
		case sig = <-ch:
			fmt.Printf("count: %d, sig: %+v\n", count, sig)
			count++
			if count > 0 {
				return
			}
		default:
			fmt.Println("Start to sleep")
			time.Sleep(500 * time.Millisecond)
			fmt.Println("Wake up")
		}
	}
}
