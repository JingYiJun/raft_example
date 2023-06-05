package main

import (
	"fmt"
	"github.com/spf13/pflag"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"raft_example/raft"
)

func main() {
	var port int
	var serverId int
	var peerAddrs []string

	pflag.IntVar(&port, "port", 0, "server port")
	pflag.IntVar(&serverId, "server-id", 0, "server id")
	pflag.StringSliceVar(&peerAddrs, "peer-addrs", nil, "peer addresses")

	pflag.Parse()

	// init peers
	var peers []*raft.Peer
	for _, addr := range peerAddrs {
		log.Printf("peer address: %s", addr)
		client, err := rpc.DialHTTP("tcp", addr)
		if err != nil {
			log.Fatalf("rpc.DialHTTP error: %v", err)
		}

		args := struct{}{}
		reply := &raft.ServerInfo{}
		err = client.Call("Raft.GetInfo", args, reply)
		if err != nil {
			log.Fatalf("client.Call error: %v", err)
		}

		peers = append(peers, &raft.Peer{
			Id:     reply.Id,
			Client: client,
		})
	}

	// init server
	rt := raft.Make(peers, serverId)

	// start server
	err := rpc.Register(rt)
	if err != nil {
		log.Fatalf("rpc.Register error: %v", err)
	}

	rpc.HandleHTTP()
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("net.Listen error: %v", err)
	}

	log.Printf("server %d listening on port %d", serverId, port)

	go http.Serve(l, nil)

	// register self to all peers
	for _, peer := range peers {
		log.Printf("registering to peer %d", peer.Id)
		args := &raft.RegisterArgs{
			Id:   serverId,
			Addr: fmt.Sprintf("localhost:%d", port),
		}
		reply := &raft.RegisterReply{}
		err := peer.Client.Call("Raft.Register", args, reply)
		if err != nil {
			log.Fatalf("peer.Client.Call error: %v", err)
		}
	}

	<-make(chan struct{})
}
