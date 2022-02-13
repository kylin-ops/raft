package raft

import (
	"fmt"
	"github.com/kylin-ops/raft/http/httpserver"
	"github.com/kylin-ops/raft/logger"
	"github.com/kylin-ops/raft/raft"
)



func StartRaft(id, addr string, port int, logg logger.Logger, members ...raft.Member) {
	r := raft.Raft{Address: fmt.Sprintf("%s:%d", addr, port), Role: "candidate", Id: id, Timeout: 5}
	if logg == nil {
		r.Logger = &logger.Log{}
	}
	var _members = map[string]*raft.Member{}
	for _, member := range members {
		_members[member.Id] = &raft.Member{Address: member.Address, Id: member.Id}
	}
	r.Members = _members
	raft.RaftInstance = &r
	go httpserver.StartHttpServer(addr, port, r.Logger)
	go r.BackendElection()
	go r.BackendHeatbeat()
	go r.BackendReCandidate()
	select {}
}
