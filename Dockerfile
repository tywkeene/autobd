FROM ubuntu

MAINTAINER Tyrell Keene <tyrell.wkeene@gmail.com>

RUN apt-get update 
RUN apt-get install -y golang-go git
RUN mkdir /root/go

ENV GOPATH=/root/go
RUN go get github.com/SaviorPhoenix/autobd

WORKDIR /root/go/src/github.com/SaviorPhoenix/autobd/
RUN go build -v -x -ldflags "-X main.commit $(git rev-parse --short=10 HEAD)"

WORKDIR /

EXPOSE 8081

ENTRYPOINT ./root/go/src/github.com/SaviorPhoenix/autobd/autobd -root=$ROOTDIR
