.PHONY: all clean dist

NAME=docker-supervise
VERSION=0.00

UNAME_S := $(shell uname -s)

all: bin

bin: *.go
ifeq ($(UNAME_S), Linux)
	go build -o build/docker-supervise
endif
ifeq ($(UNAME_S), Darwin)
	GOOS=linux GOARCH=amd64 go build -o build/docker-supervise
	go build -o build/docker-supervise-darwin
endif

clean:
	rm -rf build
	rm -rf *~

dist: clean
	git archive --format=tar --prefix=$(NAME)-$(VERSION)/ HEAD | xz -9v > $(NAME)-$(VERSION).tar.xz

container: bin Dockerfile
	docker build --no-cache -t docker-supervise .
	touch build/container