package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

// the format in which we store timings for our successful POST requests
// if we stored more timings, this might have more fields like traceId, operation, etc
type durationTableEntry struct {
	id         string
	duration   time.Duration
	startTime  time.Time
	finalState string
}

// the table in which we store the timings of our successful POST requests
// I suspect that there might be a more go-ful way to do this but this is what I went with for the demo
// my thinking is that we'll need both read (composing the status message) and write (adding new entries)
// access to this so a mutex makes sense
type durationLogT struct {
	mu    sync.Mutex
	table []durationTableEntry
}

var durLog durationLogT

// the contents of our status messsages ready for json serialization
type statusMessageT struct {
	Total   int64 `json:"total"`
	Average int64 `json:"average"`
}

func logDuration(entryArg *durationTableEntry) {
	entry := *entryArg
	entry.duration = time.Now().Sub(entry.startTime)

	// the spec was unclear; I'm choosing to only log durations of successful request
	// see readme for details
	if entry.finalState == "ok" {
		durLog.mu.Lock()
		durLog.table = append(durLog.table, entry)
		durLog.mu.Unlock()
	}
	// note: in testing, the average duration was often zero - it seems unlikely but
	// if there was an error, I didn't find it
}

// here we handle requests to create a new hash
func handleHashCreate(waitgroup *sync.WaitGroup) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		// storing a pointer so the defer call has the latest values
		status := &durationTableEntry{startTime: time.Now(), finalState: "unknown"}
		defer logDuration(status)

		if req.Method != "POST" {
			http.Error(w, "the hash endpoint is POST only", http.StatusMethodNotAllowed)
			status.finalState = "failed:methodNotAllowed"
			return
		}

		defer req.Body.Close()

		// a 2048 byte password seems pathological - limit here
		req.Body = http.MaxBytesReader(w, req.Body, 2048)
		buf := new(strings.Builder)
		_, err := io.Copy(buf, req.Body)
		if err != nil {
			http.Error(w, "Post body too large", http.StatusBadRequest)
			status.finalState = "failed:bodyTooLarge"
			return
		}
		passwd := buf.String()
		var command string
		if i := strings.IndexByte(passwd, '='); i >= 0 {
			command = passwd[:i]
			passwd = passwd[i+1:]
			if command != "password" {
				// tradeoff between user friendly and security-by-obscurity - I've chosen the latter
				// depending on the client, it might be better to return a more helpful error message
				http.Error(w, "post not in expected format", http.StatusBadRequest)
				status.finalState = "failed:postBodyBadlyFormed:badKey"
				return
			}
		} else {
			http.Error(w, "post not in expected format", http.StatusBadRequest)
			status.finalState = "failed:postBodyBadlyFormed:noKeyValSeparator"
			return
		}

		// the meat of this request: asking the hasher to hash the password
		var hasherReqId string
		waitgroup.Add(1) // making sure we don't exit before this background task finishes
		if hasherReqId, err = hashCreationRequest(passwd, waitgroup); err != nil {
			errMsg := fmt.Sprintf("hasher failed: %s", err)
			http.Error(w, errMsg, http.StatusInternalServerError)
			status.finalState = "failed:hasherReturnedError"
			return
		}
		status.id = hasherReqId
		fmt.Fprint(w, hasherReqId)
		status.finalState = "ok"
		return
	}
}

func handleHashRead(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "specifying a hashid is a GET only operation", http.StatusMethodNotAllowed)
		return
	}

	// being pretty strict on api use
	// we don't want to let this url work: /hash/foo/1
	// using a more refined request router would remove the need for this and some of the other
	// correctness tests in these functions
	dir, id := path.Split(req.URL.Path)
	if dir != "/hash/" {
		http.Error(w, "mal formed request url", http.StatusBadRequest)
		return
	}

	// as of this writing, the hasher only returns an error when the key isn't found
	// if the hasher were something more sophisticated, we'd have a richer diagnostic
	if hash, err := hashRead(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else {
		fmt.Fprintf(w, hash)
	}
}

func showStats(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "stats GET only operation", http.StatusMethodNotAllowed)
		return
	}

	var totalDur int64
	var totalEntries int64
	durLog.mu.Lock()
	for _, entry := range durLog.table {
		totalDur += entry.duration.Microseconds()
		totalEntries += 1
	}
	durLog.mu.Unlock()

	var status statusMessageT

	// the spec called for a number of microseconds as an int - else I might have returned
	// a float
	if totalEntries > 0 {
		status.Average = totalDur / totalEntries
		status.Total = totalEntries
	}

	if result, err := json.Marshal(&status); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
	} else {
		fmt.Fprint(w, string(result))
	}

}

func shutdownHandler(donechannel chan bool) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "initiating shutdown")
		donechannel <- true
	}
}
