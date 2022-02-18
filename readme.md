# 1 功能
根据raft协议进行多个节点运行是选取leader，有leader向所有成员发生心跳信息

- 核心配置： 参考example/main.go 配置成员并启动服务
- 重要配置：<br />
    NoElection       本节点不参与投票 <br />
	DefaultLeader    设置默认leader，当值和节点ID一致时，启动后默认就是leader<br />
	HealthChecker    节点健康检查接口，返回error时节点不正常<br />

# 2 使用范例
```go
package main

import (
	"flag"

	"github.com/kylin-ops/raft"
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
	})
	r.Start()
}

```