package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"os/exec"

	"github.com/codegangsta/martini"
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

	m := martini.Classic()
	m.Any("/:user/:repo/**", gitMasquerade)

	http.Serve(l, m)
}

func gitMasquerade(rw http.ResponseWriter, r *http.Request) {
	log.Printf("executing git command for url %q", r.URL.Path)

	cmd := exec.Command("git", "http-backend")
	cmd.Env = []string{
		fmt.Sprintf("GIT_PROJECT_ROOT=%s", *gitdir),
		fmt.Sprintf("GIT_HTTP_EXPORT_ALL="),
		fmt.Sprintf("PATH_INFO=%s", r.URL.Path),
		fmt.Sprintf("QUERY_STRING=%s", r.URL.RawQuery),
		fmt.Sprintf("REQUEST_METHOD=%s", r.Method),
		fmt.Sprintf("CONTENT_TYPE=%s", r.Header.Get("Content-Type")),
	}

	log.Printf("Using env %+v", cmd.Env)

	buffer := &bytes.Buffer{}

	// copy the output to the connection
	cmd.Stdin = r.Body
	cmd.Stdout = buffer
	cmd.Stderr = os.Stderr

	// run the command
	err := cmd.Run()

	if err != nil {
		log.Printf("error running http-backend err=%q", err)
		rw.WriteHeader(501)
		fmt.Fprintf(rw, "error running http-backend err=%q", err)
		return
	}

	text := textproto.NewReader(bufio.NewReader(buffer))

	code, _, _ := text.ReadCodeLine(-1)

	if code != 0 {
		rw.WriteHeader(code)
	}

	headers, _ := text.ReadMIMEHeader()

	for key, values := range headers {
		log.Printf("setting header %s => %v", key, values)
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}

	io.Copy(rw, text.R)
}
