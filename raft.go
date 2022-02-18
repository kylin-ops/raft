package raft

import (
	"encoding/json"
	"sync"

	"github.com/kylin-ops/raft/logger"
	"github.com/kylin-ops/raft/raft/health"
)

var RaftInstance *Raft

type Options struct {
	Id            string             `json:"id"`             // 本节点id
	Port          int                `json:"port"`           // 集群通信地址
	Members       map[string]*Member `json:"members"`        // raft集群成员
	Timeout       int64              `json:'timeout'`        // 多少秒没有收到心跳置为超时
	NoElection    bool               `json:"no_election"`    // 本节点是否不参加leader选举
	DefaultLeader string             `json:"default_leader"` // 默认leader
	HealthChecker health.Checker     `json:"-"`
	Logger        logger.Logger      `json:"-"` // 日志接口
}

type Leader struct {
	Term     int64  `json:"trem"`
	LeaderId string `json:"leader_id"`
}

type Member struct {
	Id                string `json:"id"`
	Address           string `json:"address"`
	Role              string `json:"role"`
	LeaderId          string `json:"leader_id"`
	ElectionStatus    string `json:"election_status"`     // 选举状态ok时，竞选者不在发送选举信息
	HeartbeatStatus   string `json:"heartbeat_status"`    // 心跳检测状态
	LastHeartbeatTime int64  `json:"last_heartbeat_time"` // 最后一次接收时间
	//Term              int64  `json:"term"`              // leader 发生任期信息
}

type HeartbeatBody struct {
	Leader  string             `json:"leader"`
	Members map[string]*Member `json:"members"`
}

// Raft 声明raft
type Raft struct {
	Options
	Mu                sync.Mutex `json:"-"`                   //锁
	LastHeartbeatTime int64      `json:"last_heartbeat_time"` // 最后一次更新时间
	VotedFor          string     `json:"voted_for"`           // 为那个节点投票  "" 代表没投票
	VotedCount        int        `json:"voted_count"`         // 获得的票数
	Role              string     `json:"role"`                // 0 follower  1 candidate  2 leader
	CurrentLeader     string     `json:"current_leader"`      // 集群当前的leader

	// Id            string             `json:"id"`
	// Address       string             `json:"address"`
	// Members       map[string]*Member `json:"members"`        // 所有成员
	// Timeout       int64              `json:"timeout"`        // 超时时间心跳超过多少时间需要初始化重新选举
	// NoElection    bool               `json:"NoElection"`     // 不参与leader选举
	// DefaultLeader string             `json:"default_leader"` //
	// HealthChecker health.Checker     `json:"-"`
	// Logger        logger.Logger      `json:"-"`

}

func (r *Raft) GetMembers() map[string]*Member {
	var members map[string]*Member
	d, _ := json.Marshal(r.Members)
	_ = json.Unmarshal(d, &members)
	return members
}

func (r *Raft) Start() {
	//go httpserver.StartHttpServer("0.0.0.0", r.Port, r.Logger)
	go r.BackendElection()
	go r.BackendHeatbeat()
	go r.BackendReCandidate()
	go r.BackendDefaultLeader()
	select {}
}

func NewRaft(o *Options) *Raft {

	return &Raft{
		Options: *o,
	}
}
