# Raft Example

A raft implementation of MIT 6.824 lab2A and part of lab2B

Include a server and client example communicating with `net/rpc`

## Usage

Server side

```shell
# start the first server and bind 8000 port
go run server/main.go --port 8000 --server-id 1
# start the second server in separate terminal and bind 8001 port, communicate with server 1
go run server/main.go --port 8001 --server-id 2 --peer-addrs 127.0.0.1:8000
# start the third server in another separate terminal and bind 8002 port, communicate with server 1,2
go run server/main.go --port 8002 --server-id 3 --peer-addrs 127.0.0.1:8000,127.0.0.1:8001
```

Client side

```shell
# start client, then use the interactive cli
go run client/main.go

> get -a 127.0.0.1:8000 # get the value stored in server 1, and if it is the leader
> set -a 127.0.0.1:8001 -v 5 # set the value stored in server 2 to 5
```