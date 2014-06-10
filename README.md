docker-supervise
================

Monitors IDs and automatically restarts those that die.

Usage:

	`MONITOR`: environment variable of initial IDs, seperated by ':'
	`DOCKER`: path to docker socket/port/etc

HTTP API:

    `/`: get all containers being monitored (JSON array)
    `/{id}`:
        PUT: add ID to the list of containers being monitored
        GET: check if container is being monitored (YES or NO)
        DELETE: do not monitor this container
