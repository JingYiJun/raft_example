package raft

import (
	"log"
	"math/rand"
	"time"
)

type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

// startElection starts a new election
func (rf *Raft) startElection() {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	log.Printf("Server %v start election", rf.me)

	rf.state = CANDIDATE
	rf.currentTerm++
	rf.votedFor = rf.me
	rf.votes = 1
	rf.resetElectionTimer()

	// Send RequestVote RPCs to all other servers
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: rf.log[len(rf.log)-1].Index,
		LastLogTerm:  rf.log[len(rf.log)-1].Term,
	}

	for i := range rf.peers {
		go func(i int) {
			rf.sendRequestVote(i, args)
		}(i)
	}
}

// sendRequestVote sends a RequestVote RPC to a server
func (rf *Raft) sendRequestVote(server int, args RequestVoteArgs) {
	reply := &RequestVoteReply{}

	log.Printf("sendRequestVote: server %v send request to server %v, args: %v", rf.me, server, args)
	err := rf.peers[server].Call("Raft.RequestVote", args, reply)
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if err != nil {
		log.Printf("sendRequestVote error: %v", err)

		// remove the server from peers
		_ = rf.peers[server].Close()
		delete(rf.peers, server)
		delete(rf.nextIndex, server)
		delete(rf.matchIndex, server)
		rf.peerLen--
		return
	}

	if err != nil || rf.state != CANDIDATE || rf.currentTerm != args.Term {
		return
	}

	// If RPC request or response contains term T > currentTerm:
	// set currentTerm = T, convert to follower
	if reply.Term > rf.currentTerm {
		log.Printf("sendRequestVote: server %v term is %v, current term is %v, convert to follower",
			server, reply.Term, rf.currentTerm)
		rf.currentTerm = reply.Term
		rf.state = FOLLOWER
		rf.votedFor = -1
		rf.resetElectionTimer()
		return
	}

	// If votedFor is null or candidateId, and candidate’s log is at
	// least as up-to-date as receiver’s log, grant vote
	if reply.VoteGranted {
		log.Printf("sendRequestVote: server %v vote granted", server)
		rf.votes++
		if rf.state != LEADER && rf.votes > rf.peerLen/2 {
			log.Printf("sendRequestVote: [%v] become leader", rf.me)
			rf.state = LEADER
			for i := range rf.peers {
				rf.nextIndex[i] = len(rf.log)
				rf.matchIndex[i] = 0
			}
			rf.broadcastAppendEntries()
		}
	}

	return
}

// RequestVote RPC handler
func (rf *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	log.Printf("RequestVote: server %v receive request from server %v, args: %v", rf.me, args.CandidateId, args)

	// Reply false if term < currentTerm
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return nil
	}

	// If RPC request or response contains term T > currentTerm:
	// set currentTerm = T, convert to follower
	if args.Term > rf.currentTerm {
		rf.setNewTerm(args.Term)
	}

	// If votedFor is null or candidateId, and candidate’s log is at
	// least as up-to-date as receiver’s log, grant vote
	if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) &&
		(args.LastLogTerm > rf.log[len(rf.log)-1].Term ||
			(args.LastLogTerm == rf.log[len(rf.log)-1].Term &&
				args.LastLogIndex >= rf.log[len(rf.log)-1].Index)) {
		rf.votedFor = args.CandidateId
		reply.Term = rf.currentTerm
		reply.VoteGranted = true
		rf.resetElectionTimer()
		return nil
	}

	// Reply false
	reply.Term = rf.currentTerm
	reply.VoteGranted = false

	return nil
}

// resetElectionTimer resets the election timer
func (rf *Raft) resetElectionTimer() {
	rf.electionTime = time.Now().Add(time.Duration(rand.Intn(2000)+1000) * time.Millisecond)
}
