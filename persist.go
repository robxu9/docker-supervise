package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/fsouza/go-dockerclient"
)

const (
	ModeTypicalPerm os.FileMode = 0755
)

type Persister interface {
	Add(string, *docker.Config) error
	Get(string) (*docker.Config, error)
	GetAll() (map[string]*docker.Config, error)
	Remove(string) error
}

type ConfigStore struct {
	config map[string]*docker.Config
	saver  Persister
	mutex  *sync.RWMutex
}

func NewConfigStore(p Persister) *ConfigStore {
	configStore := &ConfigStore{
		config: make(map[string]*docker.Config),
		saver:  p,
		mutex:  &sync.RWMutex{},
	}

	configStore.Load()
	return configStore
}

// Load items from the persister.
func (c *ConfigStore) Load() error {
	if c.saver == nil {
		return nil
	}

	m, err := c.saver.GetAll()
	if err != nil {
		return err
	}

	for k, v := range m {
		c.config[k] = v
	}

	return nil
}

func (c *ConfigStore) Add(name string, config *docker.Config) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.config[name] = config

	if c.saver != nil {
		if err := c.saver.Add(name, config); err != nil {
			log.Printf("persist: add error: %s", err)
		}
	}
}

func (c *ConfigStore) Copy() map[string]*docker.Config {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	m := make(map[string]*docker.Config)

	for k, v := range c.config {
		m[k] = v
	}

	return m
}

func (c *ConfigStore) Get(name string) (*docker.Config, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	config, ok := c.config[name]
	return config, ok
}

func (c *ConfigStore) Remove(name string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.config, name)
	if c.saver != nil {
		if err := c.saver.Remove(name); err != nil {
			log.Printf("persist: remove error: %s", err)
		}
	}
}

type DirectoryPersister string

func (d DirectoryPersister) Filename(name string) string {
	return string(d) + "/" + name + ".json"
}

func (d DirectoryPersister) Add(name string, config *docker.Config) error {
	marshal, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(d.Filename(name), marshal, ModeTypicalPerm)
}

func (d DirectoryPersister) Get(name string) (*docker.Config, error) {
	bte, err := ioutil.ReadFile(d.Filename(name))
	if err != nil {
		return nil, err
	}

	var cfg docker.Config
	err = json.Unmarshal(bte, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (d DirectoryPersister) GetAll() (map[string]*docker.Config, error) {
	files, err := ioutil.ReadDir(string(d))
	if err != nil {
		return nil, err
	}

	m := make(map[string]*docker.Config)

	for _, v := range files {
		name := v.Name()
		name = name[:strings.LastIndex(name, ".json")]
		conf, err := d.Get(name)
		if err == nil {
			m[name] = conf
		} else {
			log.Printf("[warn] couldn't load %s: %v", name, err)
		}
	}

	return m, nil
}

func (d DirectoryPersister) Remove(name string) error {
	return os.Remove(d.Filename(name))
}
