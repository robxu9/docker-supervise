FROM busybox
MAINTAINER Robert Xu <robxu9@gmail.com>

ADD ./build/docker-supervise /bin/docker-supervise

ENV DOCKER_HOST unix:///tmp/docker.sock

VOLUME ["/mnt/data"]
WORKDIR /mnt/data

EXPOSE 8080

ENTRYPOINT ["/bin/docker-supervise"]
CMD []
