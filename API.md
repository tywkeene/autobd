# Autobd's HTTP API

## /version
Description: Returns a JSON encoded structure describing the version of the autobd server. 
Autobd nodes use this endpoint to ensure version equality with the server.

Arguments: None

Example:
```
https://host:8080/version
```
Returns:
```
{
    "server": "0.0.4",
    "commit": "8d913b8990"
}
```

## /index
Description: Returns a JSON encoded structure describing the files and directory tree on the server

Arguments:

``` dir=<requested directory to index> ``` The directory to index

``` uuid=<registered node UUID> ``` The node requesting the index, must already be identified on the server

Example: 
```
http://host:8080/v0/index?dir=/&uuid=a468d5d0-56b8-4b0d-be2f-08b7d612b055
```
Returns:
```
{
    "directory1": {
      "name": "directory1",
      "size": 4096,
      "lastModified": "2016-09-26T16:46:05.468167071-06:00",
      "fileMode": 2147484141,
      "isDir": true
    },
    "directory2": {
      "name": "directory2",
      "size": 4096,
      "lastModified": "2016-09-26T16:46:07.918183525-06:00",
      "fileMode": 2147484141,
      "isDir": true
    },
    "directory3": {
      "name": "directory3",
      "size": 4096,
      "lastModified": "2016-09-28T19:13:49.163346347-06:00",
      "fileMode": 2147484141,
      "isDir": true,
      "files": {
        "directory3/file": {
          "name": "directory3/file",
          "checksum": "8d0a77a2685b1c3781de27043f71e487a0fd8472ce08917959fc6819bd32e81a636e5f817a948fa24f6f1427978dbaeb01a26a9f214aafd10ca379086bfc3ab1",
          "size": 1002,
          "lastModified": "2016-09-29T20:54:47.516394557-06:00",
          "fileMode": 420,
          "isDir": false
        }
      }
    }
  }
```

## /sync
Description: Returns the requested file (gzip'd, if the node-side can handle it) or a directory, (tarballed and gzip'd if the node-side can handle it)


Arguments: 

``` grab=<file or directory path> ``` The file or directory to transfer

``` uuid=<registered node UUID> ``` The node requesting the sync, must already be identified on the server

Example:
```
http://host:8080/v0/sync?grab=/directory3&uuid=a468d5d0-56b8-4b0d-be2f-08b7d612b055
```

Returns: File or directory contents

## /identify
Description: Allows nodes to identify and register a UUID and node version with a server

Arguments:

``` version=<node version> ``` The version of the node (i.e 0.0.4). this ensures API compatibility, as the node will panic if server version does not match

``` uuid=<registered node UUID> ``` The node UUID this node will be identified by

Example:
```
http://host:8080/v0/identify?uuid=a468d5d0-56b8-4b0d-be2f-08b7d612b055&version=0.0.4
```

Returns: Nothing

## /nodes
Description: Returns a JSON encoded list describing the nodes registered with the server

Arguments:

```uuid=<registered node UUID>``` The node requesting the node list, must already be identified on the server


Example:
```
http://host:8080/v0/nodes
```


## /heartbeat
Description: Updates the node's status on the server

Arguments:

```uuid=<registered node UUID>``` The node requesting to be updated, must already be identified on the server

Example:
```
http://host:8080/v0/heartbeat?uuid=a468d5d0-56b8-4b0d-be2f-08b7d612b055
```

Returns: Nothing

