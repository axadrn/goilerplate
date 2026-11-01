package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
)

const highestPort = 65535

func main() {
	start := flag.Int("start", 0, "first port to try")
	port := flag.Int("port", 0, "specific port to use")
	flag.Parse()

	if flag.NArg() != 0 || (*start == 0) == (*port == 0) {
		fmt.Fprintln(os.Stderr, "devport: provide exactly one of start or port")
		os.Exit(2)
	}
	if *port != 0 {
		if *port < 1 || *port > highestPort {
			fmt.Fprintln(os.Stderr, "devport: port must be between 1 and 65535")
			os.Exit(2)
		}
		fmt.Print(*port)
		return
	}
	if *start < 1 || *start > highestPort {
		fmt.Fprintln(os.Stderr, "devport: start must be between 1 and 65535")
		os.Exit(2)
	}

	for port := *start; port <= highestPort; port++ {
		listener, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(port)))
		if err != nil {
			continue
		}
		if err := listener.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "devport: release port %d: %v\n", port, err)
			os.Exit(1)
		}
		fmt.Print(port)
		return
	}

	fmt.Fprintf(os.Stderr, "devport: no available port found at or above %d\n", *start)
	os.Exit(1)
}
