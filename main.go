/*
This is a demo implementation of an http service that computes and returns hashes of
passwords.

It has a small interface:

   POST /hash       creates a hash for a password found in the post body, returns an id
   GET /hash/{id}   retrieves a previously generated hash by id
   GET /stats       returns a json status message of the form: {"total":123,"average":456}
   GET /shutdown    initiates a shutdown of the service


   Hash Creation
       The hash creation request expects the body to be in this format, case-sensitive:
           password={string}

       It returns an id that, as of this writing, is a simple int.

   Status
       The /stats endpoint will return a message like: {"total":123,"average":456}, where
       the value for "total" is the total number of successful POST requests to the service.
       the value for "average" is the average time the service took to return an id in
       response to the POST request.

   Reading a Hash
       The /hash/{id} interface returns a simple string that is the base64 encoded hash of
       the password that corresponds to the returned {id} from the hash creation endpoint.

       If the {id} does not exist in the system, a 404 Not Found is returned.


Logs are written to stdout.

The service executable accepts one command line arg, -p, to set the port on which it listens.

Productionizing Notes
    If this were a production service it would have these additional features
       * More complete logging
          - likely both request logs as well as response logs w/ a tracing id to join them
	  - possibly also a diagnostic log for messages like "starting up" or "crashing"
	  - log rotation to both limit the size of logs but also how much disk they should use
	     . maybe better to send logs to an aggregator
       * More config - it looks like there are quite a few options - I like yaml - but this
         would use the JumpCloud standard config paradigm
          - configs would tune port, timeouts, log files, log rotation policy, storage configs
	    (like if the hasher were being backed by a key val store or database)
       * A more complete status message with info like
          - start time/uptime
	  - total requests to each endpoint
	  - total errors by endpoint
	  - if the 5-second sleep was to simulate a local high-cost function, timings for that
	  - if the 5-second sleep was to simulate a backend network request, timings for that
       * Get a certificate and use only https
       * Harden against more attack vectors - like huge, possibly unending header values, etc
       * The id returned to the client might be better as a guid.  I chose a string type for the
         internal key to the hash cache for that purpose - but per the spec, it is a number
       * The hash table currently grows w/o bound.  I added a creation time to the hash table
         value struct.  Entries could expire after some time or maybe, with a little added
	 tooling, entries could be removed based on their last access time.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

func main() {
	// let's setup our wait group for graceful exit
	var waitgroup sync.WaitGroup

	listenPort := flag.Int("p", 8080, "the port on which the service will listen")
	flag.Parse()

	// used to signal the service to shutdown
	done := make(chan bool, 1)

	go hasherRequestIdFeeder()

	http.HandleFunc("/shutdown", shutdownHandler(done))
	http.HandleFunc("/hash", handleHashCreate(&waitgroup))
	http.HandleFunc("/hash/", handleHashRead)
	http.HandleFunc("/stats", showStats)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *listenPort),
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      nil,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("Hashserver Started.  Listening on port %d", *listenPort)

	<-done

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {

		cancel()
	}()

	log.Print("Hashserver stopping: shutting down http server")
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %+v", err)
	}
	log.Print("Hashserver stopping: http server exited cleanly")

	// in practice, I'd also have a timeout on this wait
	log.Print("Hashserver stopping: waiting for inflight operations")
	waitgroup.Wait()
	log.Print("Hashserver stopped cleanly.  Exiting.")
}
