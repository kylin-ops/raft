package main

import (
	"flag"
	raft2 "github.com/kylin-ops/raft"
	"github.com/kylin-ops/raft/raft"
)

func main()  {
	members := map[string]*raft.Member{
		"id-1":{Id: "id-1", Address: "127.0.0.1:8080"},
		"id-2":{Id: "id-2", Address: "127.0.0.1:8081"},
		"id-3":{Id: "id-3", Address: "127.0.0.1:8082"},
	}

	var port int
	var Id  string
	flag.IntVar(&port, "port", 8080, "服务端口号")
	flag.StringVar(&Id, "id", "id-1", "成员id")
	flag.Parse()

	raft2.StartRaft(&raft2.Options{
		Address: "0.0.0.0",
		Port: port,
		Id: Id,
		Members: members,
	})
}
