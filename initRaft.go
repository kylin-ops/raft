package raft

import (
	"strconv"

	"github.com/kylin-ops/raft/http/httpserver"
	"github.com/kylin-ops/raft/logger"
	"github.com/kylin-ops/raft/raft"
)

type Options struct {
	Id            string                  // 本节点id
	Address       string                  // 本节点的通信地址ip/域名
	Port          int                     // 本节点的通信端口
	Logger        logger.Logger           // 日志接口
	Members       map[string]*raft.Member // raft集群成员
	Timeout       int64                   // 多少秒没有收到心跳置为超时
	NoElection    bool                    // 本节点是否不参加leader选举
	DefaultLeader string                  // 默认leader
	HealthCheckType string
}

func StartRaft(option *Options) {
	if option.Timeout == 0 {
		option.Timeout = 5
	}
	if option.HealthCheckType == "" {
		option.HealthCheckType = "default"
	}

	r := raft.Raft{
		Address: option.Address + ":" + strconv.Itoa(option.Port),
		Role:    "candidate",
		// Role:       "leader",
		Id:         option.Id,
		Timeout:    option.Timeout,
		Members:    option.Members,
		NoElection: option.NoElection,
		Logger:     option.Logger,
		DefaultLeader: option.DefaultLeader,
		HealthCheckType: option.HealthCheckType,
	}
	if r.Logger == nil {
		r.Logger = &logger.Log{}
	}
	raft.RaftInstance = &r
	go httpserver.StartHttpServer("0.0.0.0", option.Port, r.Logger)
	go r.BackendElection()
	go r.BackendHeatbeat()
	go r.BackendReCandidate()
	go r.BackendDefaultLeader()
	select {}
}
