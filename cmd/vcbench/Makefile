.PHONY: osx linux clean all

all: linux

osx: main.go
	GOOS=darwin GO111MODULE=on go build -o vcbench-osx

linux: main.go
	GOOS=linux GO111MODULE=on go build -o vcbench-linux

clean: 
	-rm vcbench-osx vcbench-linux
