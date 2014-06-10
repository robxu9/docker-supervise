package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	dclient "github.com/fsouza/go-dockerclient"
)

func envopt(name, def string) string {
	if env := os.Getenv(name); env != "" {
		return env
	}
	return def
}

func main() {
	endpoint := envopt("DOCKER", "unix:///var/run/docker.sock")

	monitor := strings.Split(envopt("MONITOR", ""), ":")
	monitorSet := NewSet()
	for _, v := range monitor {
		monitorSet.Add(v)
	}

	client, err := dclient.NewClient(endpoint)
	if err != nil {
		log.Fatalf("failed to connect to docker: %s\n", err)
	}

	events := make(chan *dclient.APIEvents)

	err = client.AddEventListener(events)
	if err != nil {
		log.Fatalf("failed to add event listener: %s\n", err)
	}

	for event := range events {
		fmt.Printf("id: %s | status: %s\n", event.ID, event.Status)
		switch event.Status {
		case "die":
			container, err := client.InspectContainer(event.ID)
			if err != nil {
				log.Printf("container disappeared?! %s\n", err)
				break
			}

			if !monitorSet.Contains(container.ID) {
				if !monitorSet.Contains(container.ID[:12]) {
					// not in our monitor list; ignore
					break
				}
			}

			err = client.StartContainer(container.ID, container.HostConfig)
			if err != nil {
				log.Printf("failed to restart container... %s\n", err)
			}
		}
	}
	log.Fatalln("docker event loop closed unexpectedly")
}
