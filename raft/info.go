package raft

import (
	"log"
	"net/rpc"
)

type ServerInfo struct {
	Id    int
	Term  int
	Value int
}

func (rf *Raft) GetInfo(args struct{}, info *ServerInfo) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	info.Id = rf.me
	info.Term = rf.currentTerm
	info.Value = rf.log[len(rf.log)-1].Value
	return nil
}

type RegisterArgs struct {
	Id   int
	Addr string
}

type CommonReply struct {
	Success bool
}

type RegisterReply struct {
	Success bool
}

func (rf *Raft) Register(args RegisterArgs, reply *RegisterReply) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	log.Printf("Receive Register from id: %v, addr: %v", args.Id, args.Addr)

	rf.peers[args.Id] = NewPeer(args.Id, args.Addr)
	rf.nextIndex[args.Id] = len(rf.log)
	rf.matchIndex[args.Id] = 0
	rf.peerLen++

	reply.Success = true
	return nil
}

func NewPeer(id int, addr string) *Peer {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Fatalf("rpc.DialHTTP error: %v", err)
	}

	args := struct{}{}
	reply := &ServerInfo{}
	err = client.Call("Raft.GetInfo", args, reply)
	if err != nil {
		log.Fatalf("client.Call error: %v", err)
	}

	return &Peer{
		Id:     reply.Id,
		Client: client,
	}
}
