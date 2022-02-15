package raft

import (
	"sync"

	"github.com/kylin-ops/raft/logger"
)

var RaftInstance *Raft

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
	//Term              int64  `json:"term"`                // leader 发生任期信息
}

// Raft 声明raft
type Raft struct {
	Mu                sync.Mutex         `json:"-"` //锁
	Id                string             `json:"id"`
	Address           string             `json:"address"`
	CurrentTerm       int64              `json:"current_term"`        // 当前任期
	TempTerm          int64              `json:"temp_term"`           // candidate拉票时使用的临时任期
	LastHeartbeatTime int64              `json:"last_heartbeat_time"` // 最后一次更新时间
	VotedFor          string             `json:"voted_for"`           // 为那个节点投票  "" 代表没投票
	VotedCount        int                `json:"voted_count"`         // 获得的票数
	Role              string             `json:"role"`                // 0 follower  1 candidate  2 leader
	CurrentLeader     string             `json:"current_leader"`      // 集群当前的leader
	Timeout           int64              `json:"timeout"`             // 超时时间心跳超过多少时间需要初始化重新选举
	Members           map[string]*Member `json:"members"`             // 所有成员
	Logger            logger.Logger      `json:"-"`
	NoElection        bool               `json:"NoElection"` // 不参与leader选举
}
