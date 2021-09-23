package main

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

	var hasherReqId string
	if hasherReqId, err = hashCreationRequest(passwd); err != nil {
		errMsg := fmt.Sprintf("hasher failed: %s", err)
		http.Error(w, errMsg, http.StatusInternalServerError)
		finalState <- "fail"
		return
	}

	fmt.Fprint(w, hasherReqId)
	finalState <- hasherReqId

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

	if hash, err := hashRead(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else {
		fmt.Fprintf(w, hash)
	}
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

func shutdownHandler(donechannel chan bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "initiating shutdown")
		donechannel <- true
	}
}
