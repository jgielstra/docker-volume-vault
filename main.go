package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/calavera/docker-volume-vault/vault"
	"github.com/docker/go-plugins-helpers/volume"
	"golang.org/x/sys/unix"
)

const id = "vault"

var (
	defaultPath = filepath.Join(volume.DefaultDockerRootDirectory, id)
	root        = flag.String("root", defaultPath, "Docker volumes root directory")
	url         = flag.String("url", "", "Vault server URL")
	token       = flag.String("token", "", "Vault root token")
	insecure    = flag.Bool("insecure", false, "Skip SSL validations")

)

func main() {
	var Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	if *url == "" || *token == "" {
		Usage()
		os.Exit(1)
	}

	lockMemory()

	vault.DefaultConfig = vault.NewConfig(*url, *insecure)
	d := newDriver(*root, *token)
	h := volume.NewHandler(d)
	fmt.Println("Vault Volume Plugin Running...")
	fmt.Println(h.ServeUnix("docker", id))
}

// Locks memory, preventing memory from being written to disk as swap
func lockMemory() {
	err := unix.Mlockall(unix.MCL_FUTURE | unix.MCL_CURRENT)
	switch err {
	case nil:
	case unix.ENOSYS:
		log.Println("mlockall() not implemented on this system")
	case unix.ENOMEM:
		log.Println("mlockall() failed with ENOMEM")
	default:
		log.Fatalf("Could not perform mlockall and prevent swapping memory: %v", err)
	}
}
