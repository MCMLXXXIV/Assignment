package main

import (
	"context"
	"log"
	"net/http"
	"time"
)

func main() {
	done := make(chan bool, 1)

	go hasherRequestIdFeeder()

	http.HandleFunc("/shutdown", shutdownHandler(done))
	http.HandleFunc("/hash", handleHashCreate)
	http.HandleFunc("/hash/", handleHashRead)
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
