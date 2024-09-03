package main

import (
	"fmt"
	"local_share/server"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: program [send|receive] [IP:port]")
		fmt.Println("For sender: program send")
		fmt.Println("For receiver: program receive [IP:port]")
		return
	}

	mode := os.Args[1]

	switch mode {
	case "send":
		server.Sender()
	case "receive":
		address := ":"
		if len(os.Args) > 2 {
			address = os.Args[2]
		}
		server.Receiver(address)
	default:
		fmt.Println("Invalid mode. Use 'send' or 'receive'.")
	}
}
