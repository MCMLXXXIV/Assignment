# Hashserver

This is a demo go REST server that caches and returns SHA512 hashes for passwords POSTed to one of its endpoints.

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
The value for "average" is the cumulative average time the service took to return an ids in
response successful `POST` requests.

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

Installation consists of downloading this go source, building and optionally installing, and then
running the binary on the command line.
```sh
mkdir -p $GOPATH/src/MCMLXXXIV
cd $GOPATH/src/MCMLXXXIV
git clone https://github.com/MCMLXXXIV/Assignment
cd Assignment
go build
./hashserver -p 8080
```

### Command line args
The server has just one command line arg:
```sh
-p {port}
```
The default value for port is 8080.

## Dev Notes
There was one ambigous requirement in the spec: the duration logging.  The requirement states:

    > A GET request to /stats should return a JSON object with 2 key/value pairs. The “total”
    > key should have a value for the count of POST requests to the /hash endpoint made to the
    > server so far. The “average” key should have a value for the average time it has taken to
    > process all of those requests in microseconds.

The definition of "process" is ambigous.  The project requires the hash generation to happen out
band - and complete 5 seconds after the request.  So does "process" include the out of band part?

If this were a real application, that would be the most interesting part.  After all, the spec
says we need to return immediately so timing how fast we can return is the smaller part of the
performance question.

But since the status message is to show the count of POST requests, the average duration best
associated is the POST request durations.  I'll note that in my testing, my handling of the
POST requests returns in zero time.  I thought that was an error but printf's of time.Now()
at the top and bottom of the handler show zero delta at the nanosecond granularity.  Still
might be an error.



[Go environment]: https://golang.org/doc/install




