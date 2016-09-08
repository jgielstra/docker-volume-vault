package main

import (
	"encoding/base64"
	"io/ioutil"
	"path/filepath"
	"strings"
  "log"
	"fmt"

	"github.com/calavera/docker-volume-vault/store"
	"github.com/calavera/docker-volume-vault/vault"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/hashicorp/vault/api"
)

type driver struct {
	root  string
	token string
	store store.Store
}

func newDriver(root, token string) *driver {
	return &driver{
		root:  root,
		token: token,
		store: store.NewMemoryStore(),
	}
}

func (d *driver) Create(r volume.Request) volume.Response {
	fmt.Printf("Creating: %s\n", r)
	vol := store.NewVolume(r.Name, d.token, r.Options)
	if err := d.store.Setx(vol); err != nil {
		log.Printf("[ERR]: %v\n", err)
		return volume.Response{Err: err.Error()}
	}

	if rules, ok := r.Options["policy-rules"]; ok {
		name := r.Options["policy-name"]
		if name == "" {
			name = "docker-policy-" + r.Name
		}
		token, err := d.createPolicy(name, rules)
		if err != nil {
			log.Printf("[ERR]: %v\n", err)
			return volume.Response{Err: err.Error()}
		}
		vol.Token = token
		d.store.Set(vol)
	}
	return volume.Response{}
}

//docker volume inspect
func (d *driver) Get(r volume.Request) volume.Response {
	fmt.Printf("Get %v\n", r)
	vol, err := d.store.Get(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
  return volume.Response{Volume: &volume.Volume{Name: vol.Name, Mountpoint: d.mountpoint(vol.Name)}}
}

//docker volume ls
func (d *driver) List(r volume.Request) volume.Response {
  fmt.Printf("List %v\n", r)
	var vols []*volume.Volume
	for _, v := range d.store.List() {
		vols = append(vols, &volume.Volume{Name: v, Mountpoint: d.mountpoint(v)})
		fmt.Printf("%s\n", v)
	}
	return volume.Response{Volumes: vols}
}

//docker volume rm
func (d *driver) Remove(r volume.Request) volume.Response {
	fmt.Printf("Remove %v\n", r)
	err := d.store.Del(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}


func (d *driver) Path(r volume.Request) volume.Response {
	fmt.Printf("Path %v\n", r)
	return volume.Response{Mountpoint: d.mountpoint(r.Name)}
}

//docker run -v <r.name>:/path --driver vault ...
func (d *driver) Mount(r volume.MountRequest) volume.Response {
  fmt.Printf("Mounting %v\n",r)
	vol, err := d.store.Get(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	mount, err := vol.Mount(d.root)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{Mountpoint: mount}
}

// Exit docker container
func (d driver) Unmount(r volume.UnmountRequest) volume.Response {
	fmt.Printf("Unmounting %v\n",r)
	vol, err := d.store.Get(r.Name)
	if err != nil {
		return volume.Response{Err: err.Error()}
	}

	if vol.Mounted() {
		if err := vol.Unmount(); err != nil {
			return volume.Response{Err: err.Error()}
		}
	}

	return volume.Response{}
}

func (d *driver) mountpoint(name string) string {
	return filepath.Join(d.root, name)
}

func (d *driver) client() (*api.Client, error) {
	return vault.Client(d.token)
}

func (d *driver) createPolicy(name, policy string) (string, error) {
	log.Println("createPolicy")
	var rules []byte
	var err error
	if strings.HasPrefix(policy, "@") {
		rules, err = ioutil.ReadFile(strings.TrimPrefix(policy, "@"))
	} else {
		rules, err = base64.StdEncoding.DecodeString(policy)
	}
	if err != nil {
		return "", err
	}

	client, err := d.client()
	if err != nil {
		return "", err
	}

	if err := client.Sys().PutPolicy(name, string(rules)); err != nil {
		return "", err
	}

	req := &api.TokenCreateRequest{
		Policies: []string{name},
	}

	secret, err := client.Auth().Token().Create(req)
	if err != nil {
		return "", err
	}
	return secret.Auth.ClientToken, nil
}

func (d driver) Capabilities(r volume.Request) volume.Response {
	fmt.Printf("Capabilities ",r)
    var res volume.Response
		// `local` is per engine, `global` is accross cluster https://github.com/docker/docker/pull/22077
    res.Capabilities = volume.Capability{Scope: "local"}
    return res
}
