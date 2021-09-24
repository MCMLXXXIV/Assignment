#!/bin/bash

# I didn't find any standard testing for go REST apis so here is a collection of "manual" tests
# In practice, I'd write a test that ran in the CI pipeline in whatever format that pipeline required.

echo -n "building service... "
go build
if [ $? -eq 0 ] ; then
    echo "good build.  passed"
else
    echo "build failed.  FAIL"
    exit
fi

# start up the service
port=8080
serverLog="serverTestLog_$(date +%Y%m%d_%H%M%S).txt"
echo "starting service - stdout/stderr here: $serverLog"
./hashserver.exe -p $port > $serverLog 2>&1 &


# basic read
echo -n "test basic read - hash won't exist, expect key not found... "
temp=$(mktemp)
curl -v http://localhost:${port}/hash/1 > $temp 2>&1
echo -n "inspecting result - expect 404... "
if grep -q -q '< HTTP/1.1 404 Not Found' $temp ; then
    echo "passed"
else
    echo "FAILED"
    exit
fi
rm $temp


# add hash
echo -n "test adding a hash... "
curl -v --data "password=AngryMonkey" http://localhost:${port}/hash > $temp 2>&1
echo -n "inspecting result - expect to recieve simple string: '1'... "
if [ "X$(tail -1 $temp)" = "X1" ] ; then
    echo "passed"
else
    echo "FAILED"
    exit
fi
rm $temp


echo -n "test that immediate read fails - hash shouldn't exist yet, expect key not found... "
curl -v http://localhost:${port}/hash/1 > $temp 2>&1
echo -n "inspecting result - expect 404... "
if grep -q -q '< HTTP/1.1 404 Not Found' $temp ; then
    echo "passed"
else
    echo "FAILED"
    exit
fi
rm $temp

echo -n "sleeping six seconds then checking that I get the hash back... "
sleep 6
curl -v http://localhost:${port}/hash/1 > $temp 2>&1
echo -n "inspecting result - expect to find hash... "
# pjBiAdxDGIbbEX2rPxT3jSNFVbbpXEBOvAGNiRW9d30GdRnLMYRg4OlCYMM1spiP0YpB7BuzYkRMmkjQr3TtrA==
if [ "X$(tail -1 $temp)" = "XpjBiAdxDGIbbEX2rPxT3jSNFVbbpXEBOvAGNiRW9d30GdRnLMYRg4OlCYMM1spiP0YpB7BuzYkRMmkjQr3TtrA==" ] ; then
    echo "passed"
else
    echo "FAILED"
    exit
fi
rm $temp

echo -n "slamming service w/ 400 hash create requests... "
# this sure makes my laptop fan sing
for s in $(seq 1 400) ; do curl --data "password=AngryMonkey_${s}_$(date +%s)" http://localhost:${port}/hash > $temp 2>&1 ; done
echo -n "checking that next hash create request gets the id 402... "
curl -v --data "password=AngryMonkey_${s}_$(date +%s)" http://localhost:${port}/hash >$temp 2>&1
if [ "X$(tail -1 $temp)" = "X402" ] ; then
    echo "passed"
else
    echo "FAILED"
    exit
fi

echo -n "check the status - expect to find 402 reqests in json parsable string... "
curl -v http://localhost:${port}/stats > $temp 2>&1
result="$(tail -1 $temp)"
if [ "X$(echo $result | cut -c -23)" = 'X{"total":402,"average":' ] ; then
    echo "passed"
else
    echo "FAILED"
    exit
fi
echo "you will have to check json correctness yourself - I didn't want to assume any json parsing tools: $result"


# the service should exit until any inflight hash creation requests have been serviced
echo -n "testing for graceful shutdown... "
for s in $(seq 1 5) ; do curl --data "password=foo" http://localhost:${port}/hash >/dev/null 2>&1 ; done
shutdownTime=$(date +%s)
curl http://localhost:${port}/shutdown >$temp 2>&1
echo -n "sleeping 8 seconds for service to shutdown... "
sleep 8
# those last requests should have been 403..407 - lets check that we see key 407 in the server output
# and that the last line of the output is at least 4 seconds after we issued the command
if grep -q -q 'Adding new key, hash: \[407] 9/u6bgY2' $serverLog ; then
    echo "passed"
else
    echo "failed"
    exit
fi

echo -n "checking that last line indicates good shutdown... "
lastLine="$(tail -1 $serverLog)"
if echo $lastLine | grep -q -q 'Hashserver stopped cleanly' ; then
    echo "passed"
else
    echo "FAIL"
    exit
fi

echo -n "checking that server took at least 4 seconds to shutdown... "
lastTime=$(date -d "$(tail -1 $serverLog | cut -c -19)" +%s)
diff=$(($lastTime - $shutdownTime))
if [ $diff -ge 4 ] ; then
    echo "the server took $diff seconds to close - confirmed graceful shutdown - passed"
else
    echo "FAILED: server closed $diff seconds after shutdown - should have been more than 4"
    exit
fi

echo "Test ended.  Service shut down.  Service logs: $serverLog"
rm $temp
exit


# other random things I did to test
# some of these are actually badly formed - just grep'd them out of my history for completeness
( curl --date "password=foo" http://localhost:8080/hash & ) ; curl http://localhost:8080/shutdown
( curl http://localhost:8080/hash/1 & ) ; curl http://localhost:8080/shutdown
curl --data "passwor" http://localhost:8090/hash
curl --data "password=AngryMonkey" http://localhost:8090/hash
curl --data "password=AngryMonkey" http://localhost:8090/hash ; curl http://localhost:8090/stats
curl --data "password=AngryMonkey" http://localhost:8090/hash/2
curl --data "password=AngryMonkey2" http://localhost:8080/hash
curl --data "Password=AngryMonkey2" http://localhost:8080/hash
curl --data "password=AngryMonkey2" http://localhost:8080/hash ; sleep 1 ; curl http://localhost:8080/shutdown
curl --data "password=AngryMonkey2" http://localhost:8080/hash ; sleep 6 ; curl http://localhost:8080/hash/1
curl --data "Password=AngryMonkey2" http://localhost:8080/hash/3
curl --data "Password=AngryMonkey2" http://localhost:8080/shutdown
curl --data "Password=AngryMonkey2" http://localhost:8080/status
curl --data "password=AngryMonkey2" http://localhost:8090/hash
curl --data "Password=AngryMonkey2" http://localhost:8090/hash
curl --data "Password=AngryMonkey2" http://localhost:8090/status
curl --data "passwrd=AngryMonkey" http://localhost:8090/hash
curl --data "thing" http://localhost:8080/stats
curl --data @foo.b64 http://localhost:8080/hash
curl --data @foo.dat http://localhost:8080/hash
curl http://localhost:8080/hash/1 & ; curl http://localhost:8080/shutdown
curl http://localhost:8080/hash/asdfaskdfjalskjdfkasjdflkjasldfjaksjfd/1
curl http://localhost:8080/shutdown
curl http://localhost:8080/stats
curl http://localhost:8090/hash
curl http://localhost:8090/hash/1
curl http://localhost:8090/hash/2
curl http://localhost:8090/hash/23
curl http://localhost:8090/hash/3
curl http://localhost:8090/hash/407
curl http://localhost:8090/hash/408
curl http://localhost:8090/hash/44
curl http://localhost:8090/hash/5
curl http://localhost:8090/hash/ff
curl http://localhost:8090/hash/fflkja
curl http://localhost:8090/shutdown
curl http://localhost:8090/stats
curl http://localhost:8090/status
curl -Iv --data "password=AngryMonkey" http://localhost:8090/hash -next --data "password=FooBar" http://localhost:8090/hash
curl -Iv --data "password=AngryMonkey" http://localhost:8090/hash -next http://localhost:8090/hash
curl -Iv --data "password=AngryMonkey" http://localhost:8090/hash -next http://localhost:8090/stats 
curl -v --data "password=AngryMonkey" http://localhost:8090/hash -next --data "password=FooBar" http://localhost:8090/hash
for s in $(seq 0 400) ; do curl --data "password=$(base64 -w0 foo)" http://localhost:8080/hash ; done
for s in $(seq 0 400) ; do curl --data "password=AngryMonkey_${s}" http://localhost:8090/hash ; done

