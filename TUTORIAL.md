First off, you'll need docker if you want to run autobd in docker containers. 
Otherwise, all you need is the latest verison of Go.

## Getting autobd
```
git clone github.com/tywkeene/autobd
cd autobd/
```
Or, if you use go get
```
go get github.com/tywkeene/autobd
cd $GOPATH/src/github.com/tywkeene/autobd
```

You should now be in the autobd directory.

## Running the server

There are multiple scripts in the ```scripts/``` directory, the ones you need to worry about right now are ```scripts/deploy-server.sh```
and ```scripts/deploy-nodes.sh```.

First you will open ```scripts/deploy-server.sh``` in your favorite text editor, and look for the ```DATA_DIR``` variable.
This is the directory that autobd will index and backup. Say you want to backup your projects. You would change this value to
```/home/user/src/```. Now, when you run ```scripts/deploy-server.sh``` (from the top level of the autobd directory), this script
will start a docker container, and mount your data directory to the container.

Now you can curl the container, and it should respond:

```
$ curl "http://0.0.0.0:8080/version"
{
  "api": "0.0.6",
  "node": "0.0.6",
  "cli": "0.0.1",
  "commit": "ef710f3de1"
 }
```

Now you have an autobd server running on docker. Neat.

## The Node(s)

Now, on the machine(s) you want to backup the server files to, you will be using the ```scripts/deploy-nodes.sh``` script.
The same variable you edited in the server script must be edited. This is where the autobd node will put the files it syncs from the
server.

Once you're satisfied with your directory, you can start the node by running the ```scripts/deploy-nodes.sh``` script.
The only argument this script takes is the amount of nodes to start in paralell. e.g: ```./scripts/deploy-nodes.sh 5``` would start
5 docker containers running autobd. Of course this isn't useful for you, so you would pass the script the value 1.

## Further configuration

The configuration files allow for more altering of how autobd works. You mostly don't need to worry about these options,
since autobd runs in a docker container via a script that does everything for you. Each options is commented to help you out.
 
#### config.toml.node
```
#Which directory to
root_dir = "/home/autobd/data/"

#Run as a node
run_as_node = true

[node]
#What server to communicate with IP/URL
#(required when running as a node)
servers = ["http://172.18.0.2:8080"]

#Don't fail if the node's version doesn't match the server's
node_ignore_version_mismatch = false

#How often to update with the servers
update_interval = "30s"

#How often to request the node's status on the servers
heartbeat_interval = "15s"

#How many heartbeats the server is allowed to miss before it's ignored
max_missed_beats = 3

#Which directory on the node to sync
target_directory = "/"

#Where to store the node uuid file
uuid_path = ".uuid"
```

### config.toml.server
```
#Which directory to serve
root_dir = "/home/autobd/data/"

#The port the API will listen on
api_port = "8080"

#Use tls/ssl
use_ssl = false

#Path to the TLS certificate to use when running as server
tls_cert = "/home/autobd/secret/cert.pem"

#Path to the key associated with the TLS certificate used by the server
tls_key= "/home/autobd/secret/key.pem"

#Run as a node
run_as_node = false

#Enable the nodes endpoint (emits json metadata on all nodes)
node_endpoint = true

#How often the server will update the status of its nodes
heartbeat_tracker_interval = "30s"

#How long a node can go without a heartbeat before it's marked offline
#i.e if heartbeat_interval in config.toml.node is 15s, setting heatbeat_offline
#here to 30s would allow for a node to miss 2 heartbeats before being marked offline
heartbeat_offline = "30s"

#Where to store node metadata file
node_metadata_file = ".node_metadata"
```
