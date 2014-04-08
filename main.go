package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os/exec"
	"strings"
)

var (
	host   = flag.String("host", "localhost", "host to listen on")
	port   = flag.Int("port", 8080, "port to listen on")
	gitdir = flag.String("gitdir", "/tmp", "project root")
)

func main() {
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *host, *port)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("error listening err=%q", err)
	}

	log.Printf("listening on addr=%q", l.Addr().String())

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("error accepting err=%q", err)
			return
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	httpRequest, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("error reading line err=%q", err)
		return
	}

	results := strings.Split(httpRequest, " ")
	method := results[0]
	u := results[1]

	parsedUrl, err := url.Parse(u)
	if err != nil {
		log.Printf("error parsing url err=%q", err)
		return
	}

	log.Printf("executing git command for url %q", parsedUrl)

	cmd := exec.Command("git", "http-backend")
	cmd.Env = []string{
		fmt.Sprintf("GIT_PROJECT_ROOT=%s", *gitdir),
		fmt.Sprintf("GIT_HTTP_EXPORT_ALL="),
		fmt.Sprintf("PATH_INFO=%s", parsedUrl.Path),
		fmt.Sprintf("QUERY_STRING=%s", parsedUrl.RawQuery),
		fmt.Sprintf("REQUEST_METHOD=%s", method),
		fmt.Sprintf("GIT_PROJECT_ROOT=%s", *gitdir),
	}

	fmt.Printf("Using env %+v", cmd.Env)

	// copy the output to the connection
	cmd.Stdout = output{conn}

	// run the command
	err = cmd.Run()

	if err != nil {
		log.Printf("error running http-backend err=%q", err)
	}
}
