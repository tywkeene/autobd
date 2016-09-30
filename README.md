![autobd] (logo.png?raw=true "autobd")

[![Stories in Ready](https://badge.waffle.io/tywkeene/autobd.svg?label=ready&title=Ready)](http://waffle.io/tywkeene/autobd)
[![Build Status](https://travis-ci.org/tywkeene/autobd.svg)](https://travis-ci.org/tywkeene/autobd)
[![GoDoc](https://godoc.org/github.com/tywkeene/autobd?status.svg)](https://godoc.org/github.com/tywkeene/autobd)

## Getting autobd
Golang is required, so [get that](https://golang.org/doc/install). Most versions should work but it's probably best to
stick with at least 1.4.2 as that's what I'm developing on.

After you have go installed, it's as simple as either cloning the git repo:

`git clone https://github.com/tywkeene/autobd`

or using go get:

`go get github.com/tywkeene/autobd`

and to build

```
$ cd autobd
$ ./build.sh
```

## Getting it running

Currently only the HTTP API is in a fully working state, meaning you can `curl` to your heart's content.

To run autobd, simply do: `./autobd -root /full/path/to/directory/` (must be full path) in the autobd directory.

Autobd will listen on port `8081` by default and will be chrooted into the directory passed to -root

###Dockerfile

Autobd ships with a Dockerfile. You'll of course need [docker](https://docs.docker.com/installation/) to build and run
the image.

After you have docker up and running do `docker build -t autobd:latest .` in the autobd directory

and to run `docker run --name autobd -p 8081:8081 -v /path/to/data:/home/autobd/data autobd:latest`

This will run the autobd docker image you just built in a container called `autobd` with the data directory you passed
to the `-v` flag.

Autobd will listen for connections on the port specified on the left side of the `:` in the `-p` flag.
i.e `-p 123:8081` will cause the container to listen on port `123`.

Same goes for the `-v` flag: `-v /a/b/c:/data` will mount `/a/b/c` (on the host) to `/data/` inside the container

## The API
See [API.md](https://github.com/tywkeene/autobd/blob/master/API.md)

## Contributing

Open a pull request or issue with your idea or bug. Really. Anything wrong with anything anywhere, open a ticket and let me know,
then we can work on it together :) (Just be sure to check the [story board](https://waffle.io/tywkeene/autobd) before creating a new ticket)
