package raft

import "log"

type Log struct {
	Term  int
	Index int
	Value int
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []Log
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

// broadcastAppendEntries sends AppendEntries RPCs to all other servers
func (rf *Raft) broadcastAppendEntries() {
	if rf.peerLen == 0 {
		log.Printf("[%v]: no peer", rf.me)
		return
	}
	log.Printf("[%v]: broadcastAppendEntries", rf.me)
	for i := range rf.peers {
		go func(i int) {
			rf.sendAppendEntries(i)
		}(i)
	}
}

// sendAppendEntries sends a AppendEntries RPC to a server
func (rf *Raft) sendAppendEntries(server int) {
	log.Printf("sendAppendEntries: [%v] send request to [%v]", rf.me, server)

	// If last log index ≥ nextIndex for a follower: send AppendEntries RPC
	// with log entries starting at nextIndex
	args := AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: rf.nextIndex[server] - 1,
		PrevLogTerm:  rf.log[rf.nextIndex[server]-1].Term,
		Entries:      rf.log[rf.nextIndex[server]:],
		LeaderCommit: rf.commitIndex,
	}

	reply := &AppendEntriesReply{}
	err := rf.peers[server].Call("Raft.AppendEntries", args, reply)

	rf.mu.Lock()
	defer rf.mu.Unlock()

	if err != nil {
		log.Printf("sendAppendEntries error: %v", err)

		// remove the server from peers
		_ = rf.peers[server].Close()
		delete(rf.peers, server)
		delete(rf.nextIndex, server)
		delete(rf.matchIndex, server)
		rf.peerLen--
		return
	}

	if err != nil || rf.state != LEADER {
		return
	}

	if reply.Term > rf.currentTerm {
		rf.currentTerm = reply.Term
		rf.state = FOLLOWER
		rf.votedFor = -1
		rf.resetElectionTimer()
		return
	}

	if reply.Term < rf.currentTerm {
		return
	}

	// If successful: update nextIndex and matchIndex for follower
	if reply.Success {
		log.Printf("sendAppendEntries success: %v", reply)
		rf.nextIndex[server] = args.PrevLogIndex + len(args.Entries) + 1
		rf.matchIndex[server] = rf.nextIndex[server] - 1
	} else {
		log.Printf("sendAppendEntries fail: %v", reply)
		// If AppendEntries fails because of log inconsistency:
		// decrement nextIndex and retry
		rf.nextIndex[server]--
	}
}

// AppendEntries RPC handler
func (rf *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	log.Printf("AppendEntries: %#v", args)

	// Reply false if term < currentTerm
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return nil
	}
	rf.resetElectionTimer()

	// If RPC request or response contains term T > currentTerm:
	// set currentTerm = T, convert to follower
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.state = FOLLOWER
	}

	// Reply false if log doesn’t contain an entry at prevLogIndex
	// whose term matches prevLogTerm
	if args.PrevLogIndex > len(rf.log)-1 ||
		rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return nil
	}

	// If an existing entry conflicts with a new one (same index
	// but different terms), delete the existing entry and all that
	// follow it
	if len(rf.log)-1 > args.PrevLogIndex+len(args.Entries) &&
		rf.log[args.PrevLogIndex+len(args.Entries)].Term !=
			args.Entries[len(args.Entries)-1].Term {
		rf.log = rf.log[:args.PrevLogIndex+len(args.Entries)]
	}

	// Append any new entries not already in the log
	if len(rf.log)-1 < args.PrevLogIndex+len(args.Entries) {
		rf.log = append(rf.log, args.Entries[len(rf.log)-1-args.PrevLogIndex:]...)
	}

	// If leaderCommit > commitIndex, set commitIndex =
	// min(leaderCommit, index of last new entry)
	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = args.LeaderCommit
		if rf.commitIndex > len(rf.log)-1 {
			rf.commitIndex = len(rf.log) - 1
		}
	}

	reply.Term = rf.currentTerm
	reply.Success = true
	return nil
}
