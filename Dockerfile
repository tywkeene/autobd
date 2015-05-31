FROM google/golang

MAINTAINER Tyrell Keene <tyrell.wkeene@gmail.com>

RUN go get github.com/tywkeene/autobd

WORKDIR $GOPATH/src/github.com/tywkeene/autobd/
RUN bash build.sh

WORKDIR /

RUN mkdir /data
VOLUME /data

EXPOSE 8081
ENTRYPOINT ./$GOPATH/src/github.com/tywkeene/autobd/autobd -root data
