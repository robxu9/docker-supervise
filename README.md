docker-supervise
================

Monitors containers via name and automatically restarts those that die.

Building:

It is probably a good idea to use `go get -u` to retrieve this package, as it also downloads all the necessary dependencies to build it.

Usage:

`DOCKER_HOST`: path to docker socket/port/etc

HTTP API:

`/`: list of containers being monitored (JSON array of container names)
	GET: list of containers being monitored (JSON array of container names)
	POST: takes 'id':'[id or name]', and begins to monitor it.
`/{id or name}`:
    GET: get configuration of container [404 if not monitored]
    DELETE: do not monitor this container anymore

OS X Users:

I'll leave [this here](http://dave.cheney.net/2012/09/08/an-introduction-to-cross-compilation-with-go).