docker-supervise
================

Monitors IDs and automatically restarts those that die.

Building:

It is probably a good idea to use `go get -u` to retrieve this package, as it also downloads all the necessary dependencies to build it.

Usage:

	`MONITOR`: environment variable of initial IDs, seperated by ':'
	`DOCKER_HOST`: path to docker socket/port/etc

HTTP API:

    `/`: get all containers being monitored (JSON array)
    `/{id}`:
        PUT: add ID to the list of containers being monitored
        GET: check if container is being monitored (YES or NO)
        DELETE: do not monitor this container

OS X Users:

	I'll leave [this here](http://dave.cheney.net/2012/09/08/an-introduction-to-cross-compilation-with-go).