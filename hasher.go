package main

import (
	"crypto/sha512"
	b64 "encoding/base64"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// map of ids -> hash value
// protecting it w/ a mutex because it will be written to and read asynchronously
type tableOfHashesT struct {
	mu    sync.Mutex
	table map[string]hashEntry
}

type hashEntry struct {
	registrationTime time.Time // included a registration time for possible value expiration later
	value            string    // the string holding the base64 encoded hash value
}

var createHashTableOnce sync.Once
var tableInstance *tableOfHashesT

var idChan = make(chan string)

// I had to create the map - using this singleton pattern to make sure it only
// gets created once
func getTableInstance() *tableOfHashesT {
	if tableInstance == nil {
		createHashTableOnce.Do(
			func() {
				tableInstance = &tableOfHashesT{}
				tableInstance.table = make(map[string]hashEntry)
			})
	}
	return tableInstance
}

// all this does is feed the channel that provides new ids
func hasherRequestIdFeeder() {
	currentId := 1
	for {
		idChan <- fmt.Sprintf("%d", currentId)
		currentId += 1
	}
}

// this gets an id and then spawns a go routine that waits, then adds the base64
// encoded hash value to the table
func hashCreationRequest(passwd string, waitgroup *sync.WaitGroup) (hasherReqId string, err error) {
	hasherReqId = <-idChan
	go func(entryId string, password string) {
		// our caller added one to the waitgroup so we don't exit before completing - making sure to call done here
		defer waitgroup.Done()
		time.Sleep(time.Second * 5)
		b := []byte(password)
		hash := sha512.Sum512(b)
		encoded := b64.StdEncoding.EncodeToString(hash[:])
		log.Printf("Adding new key, hash: [%s] %s\n", entryId, encoded)
		tableOfHashes := getTableInstance()
		tableOfHashes.mu.Lock()
		tableOfHashes.table[entryId] = hashEntry{registrationTime: time.Now().UTC(), value: encoded}
		tableOfHashes.mu.Unlock()
	}(hasherReqId, passwd)
	return hasherReqId, nil
}

// the only error is failing to find a key - if the table were stored on disk or another server,
// the errors would be more varied - would need to revisit error handling to make sure our
// service "does the right thing"
func hashRead(id string) (hash string, err error) {
	tableOfHashes := getTableInstance()
	tableOfHashes.mu.Lock()
	hashentry, ok := tableOfHashes.table[id]
	tableOfHashes.mu.Unlock()
	if !ok {
		return "", errors.New("hasher: key not found")
	}
	return hashentry.value, nil
}
