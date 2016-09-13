# Docker Vault volume extension

## State of the project

I just wrote all this code before going to sleep last night.

## Installation

`sudo docker-volume-vault -token <your-root-vault-token> -url <url-to-vault>`

## Usage

0) Start vault
`docker run -d -p 8200:8200 --env MODE=DEV --name vault c12e/vault`

1) Create a volume
`docker volume create --driver vault --name mysecret`

2) Mount the volume with the driver in the docker container
`docker run --volume-driver vault --volume mysecret:/etc/secret alpine sh`

docker run --rm -it -v "$PWD":/go -w /go golang:1.7 go build -o docker-volume-vault github.com/calavera/docker-volume-vault

## Building
Getting source and getting started
```
mkdir -p ~/golang/src
export GODIR=~/golang/src
go get github.com/calavera/docker-volume-vault
cd $GODIR
```

For glibc linux
```
docker run --rm -it -v "$PWD":/go -w /go golang:1.7 go build -o docker-volume-vault github.com/calavera/docker-volume-vault
```

For alpine linux ( MUSL libc )
```
 docker run --rm -it -v "$PWD":/go -w /go golang:1.7-alpine  sh -c "apk add --update alpine-sdk && go build -o docker-volume-vault-alpine github.com/calavera/docker-volume-vault"
 ```

# running in docker
This is the start of a command to dockerize the volume driver


```
docker run -it --privileged --rm -v /run/docker/plugins/vault.sock:/run/docker/plugins/vault.sock -v $PWD:/golang alpine sh

apk --update add fuse ca-certificates
```


## DO NOT COMMIT

./docker-volume-vault -token developer-token-123 -url https://172.17.0.4:8200 -insecure true
docker run -d -p 8200:8200 --env MODE=DEV --name vault c12e/vault

sudo ./docker-volume-vault -insecure=true -token developer-token-123 -url https://172.17.0.4:8200
docker run --rm -it -v "$PWD":/go -w /go golang:1.7 go build -o docker-volume-vault github.com/calavera/docker-volume-vault
