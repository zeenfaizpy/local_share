package server

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func Receiver(address string) {
	fmt.Println("File Receiver Program")

	if address == ":" {
		address += DEFAULT_PORT
	}

	saveDir := "Shared"

	// Ensure save directory exists
	err := os.MkdirAll(saveDir, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating save directory:", err)
		return
	}

	// Start listening for connections
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()

	fmt.Printf("Listening on address %s. Waiting for sender...\n", address)

	// Accept connection
	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Error accepting connection:", err)
		return
	}
	defer conn.Close()

	fmt.Printf("Connected to sender: %s\n", conn.RemoteAddr().String())

	// Receive files
	receiveFiles(conn, saveDir)

	fmt.Println("File transfer completed.")
}

func receiveFiles(conn net.Conn, saveDir string) {
	reader := bufio.NewReader(conn)

	// Wait for initial ACK
	message, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(message) != "ACK" {
		fmt.Println("Error receiving initial ACK:", err)
		return
	}

	for {
		// Read file info
		fileInfo, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error reading file info:", err)
			return
		}

		fileInfo = strings.TrimSpace(fileInfo)
		if fileInfo == "ACK" {
			fmt.Println("Received final ACK. Transfer complete.")
			break
		}

		if !strings.HasPrefix(fileInfo, "FILE:") {
			fmt.Println("Invalid file info received:", fileInfo)
			continue
		}

		filePath := strings.TrimPrefix(fileInfo, "FILE:")
		fullPath := filepath.Join(saveDir, filePath)

		err = os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			continue
		}

		sizeInfo, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading size info:", err)
			return
		}
		sizeInfo = strings.TrimSpace(sizeInfo)
		if !strings.HasPrefix(sizeInfo, "SIZE:") {
			fmt.Println("Invalid size info received:", sizeInfo)
			continue
		}
		size, err := strconv.ParseInt(strings.TrimPrefix(sizeInfo, "SIZE:"), 10, 64)
		if err != nil {
			fmt.Println("Error parsing file size:", err)
			continue
		}

		// Receive file content
		file, err := os.Create(fullPath)
		if err != nil {
			fmt.Println("Error creating file:", err)
			continue
		}
		hash := md5.New()
		writer := io.MultiWriter(file, hash)

		bytesReceived, err := io.CopyN(writer, reader, size)
		if err != nil && err != io.EOF {
			fmt.Println("Error receiving file content:", err)
			file.Close()
			continue
		}
		file.Close()

		if bytesReceived != size {
			fmt.Printf("Warning: Received %d bytes, expected %d bytes\n", bytesReceived, size)
		}

		// Read MD5 hash
		md5Info, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading MD5 info:", err)
			return
		}
		md5Info = strings.TrimSpace(md5Info)
		if !strings.HasPrefix(md5Info, "MD5:") {
			fmt.Println("Invalid MD5 info received:", md5Info)
			continue
		}
		receivedMD5 := strings.TrimPrefix(md5Info, "MD5:")

		// Verify MD5 hash
		calculatedMD5 := hex.EncodeToString(hash.Sum(nil))
		if calculatedMD5 != receivedMD5 {
			fmt.Printf("MD5 verification failed for %s. Received: %s, Calculated: %s\n", filePath, receivedMD5, calculatedMD5)
		} else {
			fmt.Printf("Received and verified file: %s\n", filePath)
		}
	}
}
