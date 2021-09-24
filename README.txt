[24 Sep 2021 1105]

Added a waitgroup so that the process wouldn't exit before the hash create routine finsishes

At first I added the oposite - a cancellation.  But then I re-read the specs and it says:

     The program should wait for any pending/in-flight work to finish before exiting.

whew.  :)


[23 Sep 2021 2200]
I think it's just about done.  Things left:
   [ ] remove this file - it's not the kind of thing you keep in production code
   [ ] spell check the files
   [ ] figure out what to do for "document any configuration or run instructions"
          I'm not sure what the go standard on this is - is it in the README.md,
	  a README.txt?  Or what the go standard is for go programs released via github.
	  [ ] find a few go programs released on github and copy the style
   [ ] increase the shutdown timeout - the spec said to wait until in-flight work is
       finished but I've added a timeout to that wait.  I think it's only 5 seconds
       now - make it at least as large as the configured http server timeouts
   [ ] read the spec again - try to find any missed requirements
   [ ] check the output of "go doc -all" - my comments are incorrectly placed
          [ ] can I do something in the comments to have go doc show the run instructions?
   

oh!  I have a comment in the code that refers readers to the "readme" - make sure to
have a readme w/ the promised information or change the comment


[23 Sep 2021 1555]
I did some code re-organization - split this up into 3 file:
   1 main.go
   2 hasher.go
   3 handlersAndLogging.go

I also moved to a more go-ful way of handing getting ids for concurrent requests - using a channel rather than a mutex, etc

Next I'm going to fix the json output.


[22 Sep 2021 1815]
Minimum requirements met.  There is still a lot to do:

* re-read the effective go 
* organize the code into multiple files
* write a test - does the Go test support include setting up and testing services?
* refactor the hasher into its own... what?  I want to say its own class but I don't think
  that's the Go way.  The idea being that the enforced delay in creating the hash suggests
  a dependant service - one that may take some time to resolve - like a database write.  I
  want to abstract that out so that it looks like a dependant service in my code.
* simple yaml config?
* decide on logging paradigm
   * log to a file?  But don't do rolling.
* look at http request tracing - include if not too time consuming
* create a status struct that is serialized into json for the /status return
* refactor the handlers to a more standard format?
* get rid of all the globals?
* I should mention, but probably not implement, that if this were a true service the hash
  table would be stored on a more durable media than just in memory.
* Also, for a real service you might be worried about data overflow - that the hashtable
  grows forever w/o bound.  Could handle this w/ exiration - based on set lifetime or maybe
  based on last access.
* maybe add a extra status interface to include some of those other stats


So far, I only see one ambigous requirement: the duration logging.  The requirement states:

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
might be an error


net/http/pprof package registers http handlers that provide debugging information

 One instance is in the encoding/json package, which defines a Marshaler interface. When the JSON encoder receives a value that implements that interface, the encoder invokes the value's marshaling method to convert it to JSON instead of doing the standard conversion.
