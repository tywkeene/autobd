FROM google/golang

MAINTAINER Tyrell Keene <tyrell.wkeene@gmail.com>

RUN go get github.com/tywkeene/autobd

WORKDIR $GOPATH/src/github.com/tywkeene/autobd/
ADD build.sh $GOPATH/src/github.com/tywkeene/autobd/build.sh
ADD VERSION $GOPATH/src/github.com/tywkeene/autobd/VERSION
RUN bash build.sh

WORKDIR /

RUN mkdir /data
VOLUME /data

EXPOSE 8081
ENTRYPOINT ./$GOPATH/src/github.com/tywkeene/autobd/autobd -root data
