# Hashserver

This is a demo Go REST server that caches and returns SHA512 hashes for passwords POSTed to one of its endpoints.

It provides a few endpoints.

| Endpoint        | Description                                                             |
| ----------      | ----------                                                              |
| POST /hash      | creates a hash for a password found in the post body, returns an id     |
| GET /hash/{id}  | retrieves a previously generated hash by id                             |
| GET /stats      | returns a json status message of the form: {"total":123,"average":456}  |
| GET /shutdown   | initiates a shutdown of the service                                     |


## Details

### Hash Creation
`POST` a password to the `/hash` endpoint.  This endpoint expects the `POST` body to be in
this format, case-sensitive:
           `password={string}`

It returns an id that, as of this writing, is a simple int.

#### Example
```sh
curl --data "password=AngryMonkey" http://localhost:8080/hash
123
```
The returned string is an id by which you can retrieve the hash 5 seconds five seconds after its creation.

### Reading a Hash
The `/hash/{id}` interface returns a simple string that is the base64 encoded hash of
the password that corresponds to the returned `{id}` from the hash creation endpoint.

If the {id} does not exist in the system, a 404 Not Found is returned.

#### Example
```sh
curl http://localhost:8080/hash/123
pjBiAdxDGIbbEX2rPxT3jSNFVbbpXEBOvAGNiRW9d30GdRnLMYRg4OlCYMM1spiP0YpB7BuzYkRMmkjQr3TtrA==
```

### Status
The `/stats` endpoint will return a message like: `{"total":123,"average":456}`, where
the value for "total" is the total number of successful POST requests to the service.
The value for "average" is the cumulative average time the service took to return ids in
response to successful `POST` requests.

#### Example
```sh
curl http://localhost:8080/stats
{"total":345,"average":678}
```
### Shutdown
A `GET` request to this endpoint will cause the service to shut down.  It will immediately stop
servicing requests but it will wait for any pending hash requests to complete before exiting.

#### Example
```sh
curl http://localhost:8080/shutdown
initiating shutdown
```


## Installation and operation
Start with a working [Go environment].

Installation consists of downloading this Go source, building and optionally installing, and then
running the binary on the command line.
```sh
mkdir -p $GOPATH/src/MCMLXXXIV
cd $GOPATH/src/MCMLXXXIV
git clone https://github.com/MCMLXXXIV/Assignment
cd Assignment
go build
./hashserver -p 8080
```

### Tests
I didn't find any standard testing for Go REST apis so I included a collection of "manual" tests
in the shell script named `test.sh`.  In practice, I'd write a test that ran in the CI pipeline
in whatever format that pipeline required.


### Command line args
The server has just one command line arg:
```sh
-p {port}
```
The default value for port is 8080.

## Dev Notes
There was one ambiguous requirement in the spec: the duration logging.  The requirement states:

    > A GET request to /stats should return a JSON object with 2 key/value pairs. The “total”
    > key should have a value for the count of POST requests to the /hash endpoint made to the
    > server so far. The “average” key should have a value for the average time it has taken to
    > process all of those requests in microseconds.

The definition of "process" is ambiguous.  The project requires the hash generation to happen out
of band - and complete 5 seconds after the request.  So does "process" include the out of band part?

If this were a real application, that would be the most interesting part.  After all, the spec
says we need to return immediately so timing how fast we can return is the smaller part of the
performance question.

But since the status message is to show the count of POST requests, the average duration best
associated is the POST request durations.  I'll note that in my testing, my handling of the
POST requests returns in zero time.  I thought that was an error but printf's of time.Now()
at the top and bottom of the handler show zero delta at the nanosecond granularity.  Still
might be an error.

An additional ambiguity here: the spec says that the value for total should be the "count of POST
requests to the /hash endpoint."  In my implementation, I only counted the successful requests.
The reason being that, as written, the most interesting thing you might learn from this stat is
the size of the hash cache.  Productionizing this service would require expanding the output of the
stats endpoint - it should have counts of errors and successes to all endpoints and a separate
count of the size of the hash cache.

## Productionizing

If this were a production service it would have these additional features
* More complete logging
   * likely both request logs as well as response logs w/ a tracing id to join them
   * possibly also a diagnostic log for messages like "starting up" or "crashing"
   * log rotation to both limit the size of logs but also how much disk they should use
      * maybe better to send logs to an aggregator
* More config - it looks like there are quite a few options - I like yaml - but this  would use the JumpCloud standard config paradigm
   * configs would tune port, timeouts, log files, log rotation policy, storage configs
	    (like if the hasher were being backed by a key val store or database)
* A more complete status message with info like
   * start time/uptime
   * total requests to each endpoint
   * total errors by endpoint
   * if the 5-second sleep was to simulate a local high-cost function, timings for that
   * if the 5-second sleep was to simulate a backend network request, timings for that
* Get a certificate and use only https
* Harden against more attack vectors - like huge, possibly unending header values, etc
* The id returned to the client might be better as a guid.  I chose a string type for the
  internal key to the hash cache for that purpose - but per the spec, it is a number
* The hash table currently grows w/o bound.  I added a creation time to the hash table
  value struct.  Entries could expire after some time or maybe, with a little added
  tooling, entries could be removed based on their last access time.


[Go environment]: https://golang.org/doc/install




