package main

import (
	"context"
	"crypto/sha512"
	b64 "encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ReqIdT struct {
	mu sync.Mutex
	v  int
}

type TableOfHashesT struct {
	mu    sync.Mutex
	table map[int]HashEntry
}

type HashEntry struct {
	registrationTime time.Time
	value            string
}

var ReqId ReqIdT

var createHashTableOnce sync.Once
var tableInstance *TableOfHashesT

func getTableInstance() *TableOfHashesT {
	if tableInstance == nil {
		createHashTableOnce.Do(
			func() {
				tableInstance = &TableOfHashesT{}
				tableInstance.table = make(map[int]HashEntry)
			})
	}

	return tableInstance
}

type DurationTableEntry struct {
	duration time.Duration
	id       string
}

type DurationLogT struct {
	mu    sync.Mutex
	table []DurationTableEntry
}

var durLog DurationLogT

func handleHashCreate(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {

		http.Error(w, "the hash endpoint is POST only", http.StatusMethodNotAllowed)
		return
	}

	finalState := make(chan string, 1)

	// not counting unsuccessful request for status message
	defer setDefaultStatus(finalState)
	defer logDuration(time.Now(), finalState)

	defer req.Body.Close()
	buf := new(strings.Builder)
	_, err := io.Copy(buf, req.Body)
	if err != nil {
		fmt.Fprintf(w, "couldn't copy body")
		finalState <- "fail"
		return
	}
	passwd := buf.String()
	if i := strings.IndexByte(passwd, '='); i >= 0 {
		passwd = passwd[i+1:]
	} else {
		http.Error(w, "post not in expected format", http.StatusBadRequest)
		finalState <- "fail"
		return
	}
	ReqId.mu.Lock()
	thisId := ReqId.v + 1
	ReqId.v = thisId
	ReqId.mu.Unlock()
	fmt.Fprintf(w, "%d", thisId)
	go func(entryId int, password string) {
		time.Sleep(time.Second * 5)
		b := []byte(password)
		hash := sha512.Sum512(b)
		encoded := b64.StdEncoding.EncodeToString(hash[:])
		log.Printf("Adding new key, hash: [%d] %s\n", entryId, encoded)
		tableOfHashes := getTableInstance()
		tableOfHashes.mu.Lock()
		tableOfHashes.table[entryId] = HashEntry{registrationTime: time.Now().UTC(), value: encoded}
		tableOfHashes.mu.Unlock()
	}(thisId, passwd)

	finalState <- fmt.Sprintf("%d", thisId)

	return
}

func setDefaultStatus(finalState chan string) {
	finalState <- "default"
	close(finalState)
}

func logDuration(start time.Time, finalState chan string) {
	duration := time.Since(start)
	for s := range finalState {
		if s == "fail" {
			// error
			return
		}

		if _, err := strconv.Atoi(s); err == nil {
			// may be a little clumsy but we'll log the first parsable id that we recieve
			entry := DurationTableEntry{duration: duration, id: s}
			durLog.mu.Lock()
			durLog.table = append(durLog.table, entry)
			durLog.mu.Unlock()
			return
		}
	}
}

func handleHashRead(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "specifying a hashid is a GET only operation", http.StatusMethodNotAllowed)
		return
	}

	dir, id := path.Split(req.URL.Path)
	if dir != "/hash/" {
		http.Error(w, "mal formed request url", http.StatusBadRequest)
		return
	}

	var val int
	var err error

	if val, err = strconv.Atoi(id); err != nil {
		msg := fmt.Sprintf("bad hash get request: can't parse id as an int: %s", err)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	tableOfHashes := getTableInstance()
	tableOfHashes.mu.Lock()
	entry, ok := tableOfHashes.table[val]
	tableOfHashes.mu.Unlock()
	if ok {
		fmt.Fprintf(w, "%s", entry.value)
		return
	} else {
		http.Error(w, "key doesn't yet exist", http.StatusTooEarly)
		return
	}
}

func doShutdown(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "shutting down")

}

func showStats(w http.ResponseWriter, req *http.Request) {
	var totalDur int64
	var totalEntries int64
	durLog.mu.Lock()
	for _, entry := range durLog.table {
		totalDur += entry.duration.Microseconds()
		totalEntries += 1
	}
	durLog.mu.Unlock()

	if totalEntries > 0 {
		averageDur := totalDur / totalEntries
		fmt.Fprintf(w, "{\"total\": %d, \"average\": %d, \"totalMicroseconds\": %d}", totalEntries, averageDur, totalDur)
	} else {
		fmt.Fprint(w, "{\"total\": 0, \"average\": 0}")
	}
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
	http.HandleFunc("/hash", handleHashCreate)
	http.HandleFunc("/hash/", handleHashRead)
	http.HandleFunc("/headers", headers)
	http.HandleFunc("/stats", showStats)

	srv := &http.Server{
		Addr:    ":8090",
		Handler: nil,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Print("Hashserver Started")

	<-done

	log.Print("Hashserver stopping")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %+v", err)
	}
	log.Print("Hashserver exited cleanly")
}
