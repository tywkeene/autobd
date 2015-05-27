# autobd
Autobd is a high-level replicated filesystem daemon.

[![Stories in Ready](https://badge.waffle.io/saviorphoenix/autobd.svg?label=ready&title=Ready)](http://waffle.io/saviorphoenix/autobd)
[![Build Status](https://travis-ci.org/SaviorPhoenix/autobd.svg)](https://travis-ci.org/SaviorPhoenix/autobd)

## Getting autobd
Golang is required, so [get that](https://golang.org/doc/install). Most versions should work but it's probably best to
stick with at least 1.4.2 as that's what I'm developing on.

After you have go installed, it's as simple as either cloning the git repo:

`git clone https://github.com/SaviorPhoenix/autobd`

or using go get:

`go get github.com/SaviorPhoenix/autobd`

and to build

```
$ cd autobd
$ ./build.sh
```

Note: The build.sh script isn't required, but it populates a variable in autobd with the current commit, making it easier
for me when/if you report a bug.

## Getting it running

Currently only the HTTP API is in a fully working state, meaning you can `curl` to your heart's content.

To run autobd, simply do: `sudo ./autobd -root /path/to/directory/` in the autobd directory.

Autobd will listen on port `8081` by default and will be chrooted into the directory passed to -root

###Dockerfile

Autobd ships with a Dockerfile. You'll of course need [docker](https://docs.docker.com/installation/) to build and run
the image.

After you have docker up and running do `docker build -t autobd:latest .` in the autobd directory

and to run `docker run -p 8081:80801 -e "ROOTDIR=/data" autobd:latest`

NOTE: I'm still working on the dockerfile, so there is no volume mounting yet.

## The API

There are currently two functional endpoints to worry about

`/v0/manifest?dir=<dirname>` and `/v0/sync?grab=<filename>`

The manifest endpoint will return a json encoded recursive directory listing of the requested directory.

The sync endpoint will transfer (or tarball and transfer, in the case of directories) the requested file.

The last endpoint is `/version`, it simply returns a json encoded struct containing the version information about
the server and API.

autobd supports gzip compression and all replies are gzip'd by default.
```
$ curl -H 'Accept-Encoding: gzip' 'http://localhost:8081/v0/manifest?dir=/a' | gzip -d 
{
    "/a/d": {
      "name": "/a/d",
      "size": 4096,
      "lastModified": "2015-05-26T06:12:31.505764208Z",
      "fileMode": 2147484141,
      "isDir": true,
      "files": {
        "/a/d/p": {
          "name": "/a/d/p",
          "size": 4096,
          "lastModified": "2015-05-26T06:12:31.47243054Z",
          "fileMode": 2147484141,
          "isDir": true
        }
      }
    }
  }
  ```

## Contributing

Open a pull request or issue with your idea or bug. Really. Anything wrong with anything anywhere, open a ticket and let me know, 
then we can work on it together :) (Just be sure to check the [story board](https://waffle.io/saviorphoenix/autobd) before creating a new ticket) 

