package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// HTTP status codes constants
const (
	StatusOk      = 200
	StatusCreated = 201
	NotFound      = 404
)

// statusText returns the text explanation of the status code
func statusText(code int) string {
	switch code {
	case StatusOk:
		return "OK"
	case StatusCreated:
		return "Created"
	case NotFound:
		return "Not Found"
	default:
		return ""
	}
}

func supportedEncoding(encodings string) string {
	if strings.Contains(encodings, "gzip") {
		return "gzip"
	}

	return ""
}

// main creates a TCP connection and treats incoming requests
func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handler(conn)
	}
}

// handler applies the correct handler to respond the incoming request
func handler(c net.Conn) {
	defer c.Close()

	req, err := http.ReadRequest(bufio.NewReader(c))
	if err != nil {
		log.Fatalf("failed to read request")
		return
	}
	path := req.URL.Path

	if path == "/" {
		handleRoot(c)
	} else if strings.Contains(path, "/files/") {
		handleFiles(c, req)
	} else if strings.Contains(path, "/echo/") {
		handleEcho(c, req)
	} else if strings.Contains(path, "/user-agent") {
		handleUserAgent(c, req)
	} else {
		handleNotFound(c)
	}
}

func handleFiles(c net.Conn, req *http.Request) {
	if req.Method == "GET" {
		getFiles(c, req)
	} else if req.Method == "POST" {
		postFiles(c, req)
	}
}

// postFiles handles the response for the files path
func postFiles(c net.Conn, req *http.Request) {
	pathChunks := strings.Split(req.URL.Path, "/")
	filename := pathChunks[2]
	fpath := filepath.Join("/tmp/data/codecrafters.io/http-server-tester", filename)
	fmt.Printf("Reading file on: %s", fpath)

	content, err := io.ReadAll(req.Body)
	if err != nil {
		panic("failed to read body")
	}

	err = os.WriteFile(fpath, content, 0644)
	if err != nil {
		fmt.Printf("path: %s", fpath)
		handleNotFound(c)
		return
	}

	res := fmt.Sprintf(
		"HTTP/1.1 %d %s\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s",
		StatusCreated,
		statusText(StatusCreated),
		len(content),
		content,
	)

	c.Write([]byte(res))
}

// getFiles handles the response for the files path
func getFiles(c net.Conn, req *http.Request) {
	path := req.URL.Path

	pathChunks := strings.Split(path, "/")
	filename := pathChunks[2]

	fpath := filepath.Join("/tmp/data/codecrafters.io/http-server-tester", filename)
	fmt.Printf("Reading file on: %s", fpath)
	fileContents, err := os.ReadFile(fpath)
	if err != nil {
		fmt.Printf("path: %s", fpath)
		handleNotFound(c)
		return
	}

	res := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", StatusOk, statusText(StatusOk), len(fileContents), fileContents)

	c.Write([]byte(res))
}

// handleUserAgent handles the response for the user-agent path
func handleUserAgent(c net.Conn, req *http.Request) {
	body := req.Header.Get("User-Agent")

	res := fmt.Sprintf("HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", StatusOk, statusText(StatusOk), len(body), body)

	c.Write([]byte(res))
}

func compress(body string) string {
	var b bytes.Buffer

	enc := gzip.NewWriter(&b)
	enc.Write([]byte(body))
	enc.Close()

	return b.String()
}

// handleEcho handles the response for the echo path
func handleEcho(c net.Conn, req *http.Request) {
	path := req.URL.Path

	pathChunks := strings.Split(path, "/")
	body := pathChunks[2]

	encoding := supportedEncoding(req.Header.Get("Accept-Encoding"))
	var res string

	if len(encoding) > 0 {
		gzip := compress(body)

		res = fmt.Sprintf(
			"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Encoding: %s\r\nContent-Length: %d\r\n\r\n%s",
			StatusOk,
			statusText(StatusOk),
			encoding,
			len(gzip),
			gzip,
		)
	} else {
		res = fmt.Sprintf(
			"HTTP/1.1 %d %s\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
			StatusOk,
			statusText(StatusOk),
			len(body),
			body,
		)
	}

	c.Write([]byte(res))
}

// handleRoot handles the response for the root path
func handleRoot(c net.Conn) {
	res := fmt.Sprintf("HTTP/1.1 %d %s\r\n\r\n", StatusOk, statusText(StatusOk))

	c.Write([]byte(res))
}

// handleNotFound handles the response for the not found path
func handleNotFound(c net.Conn) {
	res := fmt.Sprintf("HTTP/1.1 %d %s\r\n\r\n", NotFound, statusText(NotFound))

	c.Write([]byte(res))
}
