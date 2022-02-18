package main

import (
	"flag"

	"github.com/kylin-ops/raft"
	"github.com/kylin-ops/raft/health"
)

func main() {
	members := map[string]*raft.Member{
		"id-1": {Id: "id-1", Address: "127.0.0.1:8080"},
		"id-2": {Id: "id-2", Address: "127.0.0.1:8081"},
		"id-3": {Id: "id-3", Address: "127.0.0.1:8082"},
	}

	var addr string
	var Id string
	var leader string
	var noElection bool
	flag.StringVar(&addr, "addr", "0.0.0.0:8080", "服务端口号")
	flag.StringVar(&Id, "id", "id-1", "成员id")
	flag.StringVar(&leader, "leader", "", "默认leader")
	flag.BoolVar(&noElection, "no_election", false, "不参加选取")
	flag.Parse()

	r := raft.NewRaft(&raft.Options{
		Id: Id,
		Address: addr,
		DefaultLeader: leader,
		NoElection: noElection,
		Members: members,
		HealthChecker: &health.Default{},
	})
	r.Start()
}
