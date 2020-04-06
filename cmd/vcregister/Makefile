.PHONY: all clean

all: vcregister-linux vcregister-osx

vcregister-linux: 
	GOOS=linux go build -o vcregister-linux

vcregister-osx:
	GOOS=linux go build -o vcregister-osx

clean:
	-rm vcregister-linux vcregister-osx
