package raft

import (
	"log"
	"net/rpc"
	"sync"
	"time"
)

type State int

const (
	FOLLOWER State = iota
	CANDIDATE
	LEADER
)

type Peer struct {
	Id      int
	Address string
	*rpc.Client
}

type Raft struct {
	mu           sync.Mutex
	peers        map[int]*Peer // server id -> server info
	peerLen      int
	me           int
	state        State
	heartBeat    time.Duration
	electionTime time.Time
	votes        int

	// Persistent state on all servers
	currentTerm int
	votedFor    int
	log         []Log

	// Volatile state on all servers
	commitIndex int
	lastApplied int

	// Volatile state on leaders
	nextIndex  map[int]int // server id -> next log index
	matchIndex map[int]int // server id -> the highest log index known to be replicated
}

// GetState returns currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	return rf.currentTerm, rf.state == LEADER
}

// Make creates a raft server
func Make(peers []*Peer, me int) *Raft {
	rf := &Raft{}

	// Initialize the server
	rf.peers = make(map[int]*Peer)
	for _, peer := range peers {
		rf.peers[peer.Id] = peer
	}
	rf.peerLen = len(peers)
	rf.me = me
	rf.state = FOLLOWER
	rf.heartBeat = 800 * time.Millisecond
	rf.resetElectionTimer()
	rf.votes = 0

	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]Log, 1)
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = map[int]int{}
	for _, peer := range peers {
		rf.nextIndex[peer.Id] = 1
	}
	rf.matchIndex = map[int]int{}

	go rf.ticker()

	return rf
}

func (rf *Raft) ticker() {
	for {
		time.Sleep(rf.heartBeat)
		rf.mu.Lock()
		if rf.state == LEADER {
			rf.broadcastAppendEntries()
		} else if time.Now().After(rf.electionTime) {
			if rf.peerLen > 0 {
				go rf.startElection()
			} else {
				log.Printf("[%v]: the only server in the cluster, no need to start election", rf.me)
				rf.resetElectionTimer()
			}
		}
		rf.mu.Unlock()
	}
}

func (rf *Raft) setNewTerm(term int) {
	if term > rf.currentTerm || rf.currentTerm == 0 {
		rf.state = FOLLOWER
		rf.currentTerm = term
		rf.votedFor = -1
		log.Printf("[%d]: set term %v\n", rf.me, rf.currentTerm)
	}
}

type StartArgs struct {
	Value int
}

func (rf *Raft) Start(args StartArgs, reply *CommonReply) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != LEADER {
		reply.Success = false
		return nil
	}

	rf.log = append(rf.log, Log{
		Term:  rf.currentTerm,
		Value: args.Value,
	})

	log.Printf("[%d]: start, log: %v\n", rf.me, rf.log)
	rf.broadcastAppendEntries()

	reply.Success = true
	return nil
}
