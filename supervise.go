package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fsouza/go-dockerclient"
)

const (
	PERSIST_DIR = "containers"
)

var (
	confStore *ConfigStore
	client    *docker.Client
)

func envopt(name, def string) string {
	if env := os.Getenv(name); env != "" {
		return env
	}
	return def
}

func main() {
	endpoint := envopt("DOCKER_HOST", "unix:///var/run/docker.sock")
	port := envopt("PORT", "8080")

	var err error
	client, err = docker.NewClient(endpoint)
	if err != nil {
		log.Fatalf("[fatal] failed to connect to docker: %s\n", err)
	}

	persistDir := envopt("PERSIST", PERSIST_DIR)

	var persister Persister = nil

	if _, err := os.Stat(persistDir); os.IsNotExist(err) {
		log.Printf("[warn] persist dir doesn't exist, not going to persist.")
	} else {
		persister = DirectoryPersister(persistDir)
	}

	confStore = NewConfigStore(persister)
	err = confStore.Load()
	if err != nil {
		log.Printf("[warn] failed to load from persist dir: %v", err)
	}

	events := make(chan *docker.APIEvents)

	// go-dockerclient issue #101
	client.AddEventListener(events)
	client.RemoveEventListener(events)
	err = client.AddEventListener(events)
	if err != nil {
		log.Fatalf("[fatal] failed to add event listener: %s\n", err)
	}

	go monitorEvents(events)

	http.DefaultServeMux.HandleFunc("/", serveHandler)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func serveHandler(rw http.ResponseWriter, r *http.Request) {
	// remove surrounding /
	path := strings.Trim(r.URL.Path, "/")

	if path == "" {
		switch r.Method {
		case "GET":
			list := make([]string, 0)
			for k, _ := range confStore.Copy() {
				list = append(list, k)
			}

			bte, err := json.MarshalIndent(list, "", "  ")
			if err != nil {
				log.Printf("[err] trying to marshal set: %s\n", err)
				http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(rw, "%s", bte)
		case "POST":
			if err := r.ParseForm(); err != nil {
				http.Error(rw, "can't parse request form: "+err.Error(), http.StatusBadRequest)
				return
			}

			name := strings.Trim(r.Form.Get("id"), "/")
			if name == "" {
				http.Error(rw, "requires id parameter for monitoring container", http.StatusBadRequest)
				return
			}

			if _, ok := confStore.Get(name); ok {
				rw.Header().Set("Location", "/"+name)
				rw.WriteHeader(http.StatusSeeOther)
				return
			}

			err := monitorContainer(name)
			if err != nil {
				http.Error(rw, "can't monitor container: "+err.Error(), http.StatusBadRequest)
				return
			}

			rw.Header().Set("Location", "/"+name)
			rw.WriteHeader(http.StatusCreated)
		default:
			http.Error(rw, "Invalid Method "+r.Method, http.StatusBadRequest)
		}
	} else {
		conf, ok := confStore.Get(path)
		if !ok {
			http.Error(rw, "not monitoring "+path, http.StatusNotFound)
			return
		}

		switch r.Method {
		case "GET":
			bte, _ := json.MarshalIndent(conf, "", "  ")
			fmt.Fprintf(rw, "%s", bte)
		case "DELETE":
			confStore.Remove(path)
			fmt.Fprintf(rw, "okay, deleted %s.", path)
			rw.WriteHeader(http.StatusOK)
		default:
			http.Error(rw, "Invalid Method "+r.Method, http.StatusBadRequest)
		}
	}
}

func monitorContainer(name string) error {
	// verify ID
	container, err := client.InspectContainer(name)
	if err != nil {
		return err
	}

	confStore.Add(strings.Trim(container.Name, "/"), container.Config)
	return nil
}

func monitorEvents(c chan *docker.APIEvents) {
	for event := range c {
		if event.Status == "die" {
			container, err := client.InspectContainer(event.ID)

			if err != nil {
				log.Printf("[wut] container disappeared?! %s\n", err)
				continue
			}

			name := container.Name[1:]

			conf, ok := confStore.Get(name)
			if !ok {
				// we're not monitoring this name
				continue
			}

			hostConfig := container.HostConfig

			// delete old container
			err = client.RemoveContainer(docker.RemoveContainerOptions{
				ID: container.ID,
			})

			if err != nil {
				log.Printf("[err] failed to delete old container: %s", err)
				return
			}

			newContainer, err := client.CreateContainer(docker.CreateContainerOptions{
				Name:   name,
				Config: conf,
			})

			if err != nil {
				log.Printf("[err] failed to create replacement container... %s\n", err)
				return
			}

			err = client.StartContainer(newContainer.ID, hostConfig)
			if err != nil {
				log.Printf("[err] failed to up replacement container... %s\n", err)
				return
			}
		}
	}
	log.Fatalln("[fatal] docker event loop closed unexpectedly")
}
