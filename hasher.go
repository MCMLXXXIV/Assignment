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

type TableOfHashesT struct {
	mu    sync.Mutex
	table map[string]HashEntry
}

type HashEntry struct {
	registrationTime time.Time
	value            string
}

var createHashTableOnce sync.Once
var tableInstance *TableOfHashesT

var idChan = make(chan string)

func getTableInstance() *TableOfHashesT {
	if tableInstance == nil {
		createHashTableOnce.Do(
			func() {
				tableInstance = &TableOfHashesT{}
				tableInstance.table = make(map[string]HashEntry)
			})
	}

	return tableInstance
}

func hasherRequestIdFeeder() {
	currentId := 1
	for {
		idChan <- fmt.Sprintf("%d", currentId)
		currentId += 1
	}
}

func hashCreationRequest(passwd string) (hasherReqId string, err error) {
	err = nil

	hasherReqId = <-idChan
	go func(entryId string, password string) {
		time.Sleep(time.Second * 5)
		b := []byte(password)
		hash := sha512.Sum512(b)
		encoded := b64.StdEncoding.EncodeToString(hash[:])
		log.Printf("Adding new key, hash: [%s] %s\n", entryId, encoded)
		tableOfHashes := getTableInstance()
		tableOfHashes.mu.Lock()
		tableOfHashes.table[entryId] = HashEntry{registrationTime: time.Now().UTC(), value: encoded}
		tableOfHashes.mu.Unlock()
	}(hasherReqId, passwd)

	return
}

func hashRead(id string) (hash string, err error) {
	err = nil
	tableOfHashes := getTableInstance()
	tableOfHashes.mu.Lock()
	hashentry, ok := tableOfHashes.table[id]
	tableOfHashes.mu.Unlock()
	if !ok {
		return "", errors.New("hasher: key not found")
	}
	return hashentry.value, nil
}
