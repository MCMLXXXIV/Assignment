### Hashserver

This is a demo go REST server that caches and returns SHA512 hashes for passwords POSTed to one of its endpoints.

It provides a few endpoints.

| POST /hash      | creates a hash for a password found in the post body, returns an id     |
| GET /hash/{id}  | retrieves a previously generated hash by id                             |
| GET /stats      | returns a json status message of the form: {"total":123,"average":456}  |
| GET /shutdown   | initiates a shutdown of the service                                     |



#### Hash Creation
       `POST` a password to the `/hash` endpoint.  This endpoint expects the `POST` body to be in
       this format, case-sensitive:
           `password={string}`

       It returns an id that, as of this writing, is a simple int.

       ##### Example
       ```sh
       curl --data "password=AngryMonkey" http://localhost:8090/hash
       123
       ```
       The returned string is an id by which you can retrieve the hash 5 seconds five seconds after its creation.

#### Reading a Hash
       The `/hash/{id}` interface returns a simple string that is the base64 encoded hash of
       the password that corresponds to the returned `{id}` from the hash creation endpoint.

       If the {id} does not exist in the system, a 404 Not Found is returned.

       ##### Example
       ```sh
       curl http://localhost:8090/hash/123
       pjBiAdxDGIbbEX2rPxT3jSNFVbbpXEBOvAGNiRW9d30GdRnLMYRg4OlCYMM1spiP0YpB7BuzYkRMmkjQr3TtrA==
       ```

#### Status
       The `/stats` endpoint will return a message like: `{"total":123,"average":456}`, where
       the value for "total" is the total number of successful POST requests to the service.
       The value for "average" is the cumulative average time the service took to return an ids in
       response successful POST requests.

       ##### Example
       ```sh
       curl http://localhost:8080/stats
       {"total":345,"average":678}

#### Shutdown
     A GET request to this endpoint will cause the service to shut down.  It will immediately stop
     servicing requests but it will wait for any pending hash requests to complete before exiting.

     ##### Example
     ```sh
     curl http://localhost:8080/shutdown
     initiating shutdown
     ```




