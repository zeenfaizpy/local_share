package main

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

const DEFAULT_PORT = "8080"

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
		callSender()
	case "receive":
		address := ":"
		if len(os.Args) > 2 {
			address = os.Args[2]
		}
		callReceiver(address)
	default:
		fmt.Println("Invalid mode. Use 'send' or 'receive'.")
	}
}

func callSender() {
	fmt.Println("File Sharing Program")

	// Get user input
	folderPath := getUserInput("Enter folder path: ")
	receiverIP := getUserInput("Enter receiver IP: ")
	receiverPort := getUserInput("Enter receiver port (default 8080): ")

	if receiverPort == "" {
		receiverPort = DEFAULT_PORT
	}

	address := net.JoinHostPort(receiverIP, receiverPort)

	// Establish connection
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

func getUserInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func sendMessage(conn net.Conn, message string) {
	_, err := conn.Write([]byte(message + "\n"))
	if err != nil {
		fmt.Println("Error sending message:", err)
	}
}

func sendFile(conn net.Conn, filePath, basePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// Calculate MD5 hash
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Println("Error calculating MD5 hash:", err)
		return
	}
	md5Hash := hex.EncodeToString(hash.Sum(nil))

	// Reset file pointer to beginning
	file.Seek(0, 0)

	// Get relative path
	relPath, _ := filepath.Rel(basePath, filePath)
	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	// Send file info
	sendMessage(conn, "FILE:"+relPath)
	sendMessage(conn, fmt.Sprintf("SIZE:%d", getFileSize(file)))

	// Send file content
	buffer := make([]byte, 1024)
	for {
		bytesRead, err := file.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading file:", err)
			}
			break
		}
		conn.Write(buffer[:bytesRead])
	}

	// Send MD5 hash
	sendMessage(conn, "MD5:"+md5Hash)

	fmt.Printf("Sent file: %s\n", relPath)
}

func getFileSize(file *os.File) int64 {
	stat, err := file.Stat()
	if err != nil {
		return 0
	}
	return stat.Size()
}

func callReceiver(address string) {
	fmt.Println("File Receiver Program")

	if address == ":" {
		address += DEFAULT_PORT
	}

	// Get user input
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

		// Ensure directory exists
		err = os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			continue
		}

		// Read file size
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
