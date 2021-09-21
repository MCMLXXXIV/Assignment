package main

import (
	"crypto/sha512"
	b64 "encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"context"
	"log"
	"time"
)

type ReqIdT struct {
	mu sync.Mutex
	v  int
}

var ReqId ReqIdT

func hello(w http.ResponseWriter, req *http.Request) {
	ReqId.mu.Lock()
	thisId := ReqId.v + 1
	ReqId.v = thisId
	ReqId.mu.Unlock()
	s := "angryMonkey"
	b := []byte(s)
	hash := sha512.Sum512(b)
	encoded := b64.StdEncoding.EncodeToString(hash[:])
	fmt.Fprintf(w, "hello - [%d] %s\n", thisId, encoded)
}

func headers(w http.ResponseWriter, req *http.Request) {
	for name, headers := range req.Header {
		for _, h := range headers {
			fmt.Fprintf(w, "%v: %v\n", name, h)
		}
	}
}

func handleHash(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		fmt.Fprintf(w, "only post supported")
		return
	}

	defer req.Body.Close()
	buf := new(strings.Builder)
	_, err := io.Copy(buf, req.Body)
	if err != nil {
		fmt.Fprintf(w, "couldn't copy body")
		return
	}

	passwd := buf.String()

	if i := strings.IndexByte(passwd, '='); i >= 0 {
		passwd = passwd[i+1:]
	} else {
		http.Error(w, "post not in expected format", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "got passwd: [%s]", passwd)

	ReqId.mu.Lock()
	thisId := ReqId.v + 1
	ReqId.v = thisId
	ReqId.mu.Unlock()
	b := []byte(passwd)
	hash := sha512.Sum512(b)
	encoded := b64.StdEncoding.EncodeToString(hash[:])
	fmt.Fprintf(w, "[%d] %s\n", thisId, encoded)

}

func doShutdown(w http.ResponseWriter, req *http.Request) {
fmt.Fprintf(w, "doShutdown stub")

}

func showStats(w http.ResponseWriter, req *http.Request) {
fmt.Fprintf(w, "showStats stub")
}

func hackChannelIntoHandler(donechannel chan bool) func(http.ResponseWriter, *http.Request) {
     return func(w http.ResponseWriter, req *http.Request) {
          fmt.Fprintf(w, "doShutdown stub")
          donechannel <- true
     }
}

func main() {
     done := make(chan bool, 1)

     shutdownHandler := hackChannelIntoHandler(done)

     http.HandleFunc("/shutdown", shutdownHandler)
     http.HandleFunc("/hash", handleHash)
     http.HandleFunc("/headers", headers)
     http.HandleFunc("/stats", showStats)

     srv := &http.Server{
     	 Addr: ":8090",
     	 Handler: nil,
     }

     go func() {
     	if err:= srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
     	   log.Fatalf("listen: %s\n", err)
     	}
     }()
     
     log.Print("Hashserver Started")

     <- done
     
     log.Print("Hashserver stopped")
     ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
     defer func() {
        cancel()
     }()

     if err := srv.Shutdown(ctx); err != nil {
          log.Fatalf("Server shutdown failed: %+v", err)
     }
     log.Print("Hashserver exited cleanly");
}
