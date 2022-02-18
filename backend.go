// 后台运行服务
package raft

import (
	"math/rand"
	"time"
)

// 后台执行选举任务
func (r *Raft) BackendElection() {
	for {
		rand.Seed(time.Now().UnixNano())
		if r.Role != "candidate" || r.NoElection {
			time.Sleep(time.Second)
			continue
		}
		time.Sleep(time.Duration(rand.Intn(150)+150) * time.Millisecond)
		r.sendElectionToALLMembers()
	}
}

// 后台有leader发生心跳信息
func (r *Raft) BackendHeatbeat() {
	for {
		if r.Role == "leader" && r.CurrentLeader == r.Id && len(r.Members)/2 < r.VotedCount {
			r.Logger.Debugf("send heatbert")
			r.sendHeartbeatToAllMembers()
		}
		// else {
		// 	r.Logger.Debugf("heartbeat - 节点%s的角色是%s,当前的leader是%s,选票有%d", r.Id, r.Role, r.CurrentLeader, r.VotedCount)
		// }
		time.Sleep(time.Second)
	}
}

func (r *Raft) BackendDefaultLeader() {
	for {
		time.Sleep(time.Second * 1)
		if r.DefaultLeader == r.Id {
			if r.Role != "leader" || r.VotedFor != r.Id {
				r.Mu.Lock()
				if r.Role != "leader" {
					r.Role = "leader"
				}
				if r.VotedFor != r.Id {
					r.VotedFor = r.Id
				}
				r.CurrentLeader = r.Id
				r.VotedCount = len(r.Members)
				r.Mu.Unlock()
			}
		}
	}
}

// leader 联系失败，重新选举
func (r *Raft) BackendReCandidate() {
	go func() {
		for {
			time.Sleep(time.Second * 1)
			if r.DefaultLeader == r.Id {
				continue
			}

			members := r.GetMembers()
			if r.Role == "candidate" {
				for _, m := range members {
					if time.Now().Unix()-m.LastHeartbeatTime > r.Timeout {
						r.Mu.Lock()
						for id := range r.Members {
							r.Members[id].ElectionStatus = ""
						}
						r.Mu.Unlock()
					}
				}
				r.VotedFor = ""
				r.VotedCount = 0
				r.Logger.Debugf("初始化 - 锁定选票超时, 节点可投新的选票")
			}
			if r.Role == "candidate" || r.LastHeartbeatTime == 0 {
				continue
			}
			if time.Now().Unix()-r.LastHeartbeatTime > r.Timeout {
				r.Mu.Lock()
				r.CurrentLeader = ""
				r.VotedFor = ""
				for id := range r.Members {
					r.Members[id].LeaderId = ""
					r.Members[id].Role = ""
					r.Members[id].ElectionStatus = ""
				}
				r.Mu.Unlock()
				r.Logger.Debugf("初始化 - 心跳超时，节点初始化重新参与选举")
			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second * 3)
			if r.Role == "candidate" || r.LastHeartbeatTime == 0 {
				continue
			}
			if time.Now().Unix()-r.LastHeartbeatTime > r.Timeout {
				r.Role = "candidate"
				r.VotedCount = 0
			}
		}
	}()
}
