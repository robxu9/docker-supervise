package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"github.com/gorilla/mux"
	"menteslibres.net/gosexy/to"

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
	port := envopt("PORT", "8080")

	client, err := dclient.NewClient(endpoint)
	if err != nil {
		log.Fatalf("failed to connect to docker: %s\n", err)
	}

	monitor := strings.Split(envopt("MONITOR", ""), ":")
	monitorSet := NewSet()
	for _, v := range monitor {
		if v == "" {
			continue
		}

		id := v

		container, err := client.InspectContainer(id)
		if err != nil {
			log.Printf("skipping no such container %s\n", id)
			continue
		}
		id = container.ID

		monitorSet.Add(v)
	}

	events := make(chan *dclient.APIEvents)

	// go-dockerclient issue #101
	client.AddEventListener(events)
	client.RemoveEventListener(events)
	err = client.AddEventListener(events)
	if err != nil {
		log.Fatalf("failed to add event listener: %s\n", err)
	}

	go func() {
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

				monitorSet.Remove(container.ID)
				monitorSet.Remove(container.ID[:12])

				hostConfig := container.HostConfig
				name := container.Name[1:]

				if strings.HasPrefix(name, "restarted_") && strings.Contains(name, "-") {
					name = name[strings.Index(name, "-")+1:]
				}

				newContainer, err := client.CreateContainer(dclient.CreateContainerOptions{
					Name:   "restarted_" + to.String(time.Now().Unix()) + "-" + name,
					Config: container.Config,
				})

				if err != nil {
					log.Printf("failed to create replacement container... %s\n", err)
					return
				}

				err = client.StartContainer(newContainer.ID, hostConfig)
				if err != nil {
					log.Printf("failed to up replacement container... %s\n", err)
					return
				}

				monitorSet.Add(newContainer.ID)
			}
		}
		log.Fatalln("docker event loop closed unexpectedly")
	}()

	// setup REST to allow adding/removing IDs dynamically
	//
	// /               ==> show all containers being monitored
	// /{id}  [PUT]    ==> add a container to monitor
	// /{id}  [GET]    ==> check if a container is being monitored
	// /{id}  [DELETE] ==> do not monitor a container
	//

	r := mux.NewRouter()

	r.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		list := make([]string, 0)
		monitorSet.Iterate(func(str string) {
			list = append(list, str)
		})

		bte, err := json.MarshalIndent(list, "", "\t")
		if err != nil {
			log.Printf("error trying to marshal set: %s\n", err)
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(rw, "%s", bte)
	})

	r.HandleFunc("/{id:[0-9a-z]+}", func(rw http.ResponseWriter, r *http.Request) {
		v := mux.Vars(r)
		id := v["id"]

		if len(id) != 12 && len(id) != 64 {
			log.Printf("got invalid length id %s\n", id)
			http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}

		// verify ID
		container, err := client.InspectContainer(id)
		if err != nil {
			log.Printf("no such id %s\n", id)
			http.Error(rw, "no such id "+id, http.StatusNotFound)
			return
		}
		id = container.ID

		switch r.Method {
		case "PUT":
			// add to set
			monitorSet.Add(id)
			fmt.Fprint(rw, "OK")
		case "GET":
			// check if we're monitoring it
			if monitorSet.Contains(id) {
				fmt.Fprint(rw, "YES")
			} else {
				fmt.Fprint(rw, "NO")
			}
		case "DELETE":
			monitorSet.Remove(id)
			fmt.Fprint(rw, "OK")
		}

	}).Methods("PUT", "GET", "DELETE")

	log.Fatal(http.ListenAndServe(":"+port, r))
}
