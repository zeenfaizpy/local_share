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
	"strings"
)

const DEFAULT_PORT = "8080"

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

	file.Seek(0, 0)

	relPath, _ := filepath.Rel(basePath, filePath)
	// Normalize path separators ( to work with windows )
	relPath = filepath.ToSlash(relPath)

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
