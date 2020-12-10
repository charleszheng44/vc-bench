.PHONY: all clean

all: image

vcregister-linux: 
	GOOS=linux go build -o vcregister-linux

vcregister-osx:
	GOOS=linux go build -o vcregister-osx

image: Dockerfile vcregister-linux
	docker build -t virtualcluster/register-amd64 . 

clean:
	-rm vcregister-linux vcregister-osx
