.PHONY: all clean dist

NAME=docker-supervise
VERSION=0.00

all: bin

bin: *.go
	go build -o build/docker-supervise

clean:
	rm -rf build
	rm -rf *~

dist: clean
	git archive --format=tar --prefix=$(NAME)-$(VERSION)/ HEAD | xz -9v > $(NAME)-$(VERSION).tar.xz

container: bin Dockerfile
	docker build --no-cache -t docker-supervise .
	touch build/container