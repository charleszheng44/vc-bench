.PHONY: osx linux clean all

all: linux osx

osx: main.go
	GOOS=darwin go build -o vcbench-osx

linux: main.go
	GOOS=linux go build -o vcbench-linux

clean: 
	-rm vcbench-osx vcbench-linux
