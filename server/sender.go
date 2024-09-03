package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

func Sender() {
	fmt.Println("Share Files")

	folderPath := getUserInput("Enter folder path: ")
	receiverIP := getUserInput("Enter receiver IP: ")
	receiverPort := getUserInput("Enter receiver port (default 8080): ")

	if receiverPort == "" {
		receiverPort = DEFAULT_PORT
	}

	address := net.JoinHostPort(receiverIP, receiverPort)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		fmt.Println("Error connecting:", err)
		return
	}
	defer conn.Close()

	// Send initial ACK
	sendMessage(conn, "ACK")

	// Walk through the folder and send files
	err = filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			sendFile(conn, path, folderPath)
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error walking through folder:", err)
		return
	}

	// Send final ACK
	sendMessage(conn, "ACK")

	fmt.Println("File transfer completed.")
}
