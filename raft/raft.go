package raft

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/kylin-ops/raft/http/httpclient/grequest"
	"github.com/kylin-ops/raft/logger"
)

var RaftInstance *Raft

type Leader struct {
	Term     int64  `json:"trem"`
	LeaderId string `json:"leader_id"`
}

type Member struct {
	Id       string `json:"id"`
	Address  string `json:"address"`
	Role     string `json:"role"`
	LeaderId string `json:"leader_id"`
	Status   string `json:"status"` // ok error
	//Term              int64  `json:"term"`                // leader 发生任期信息
	LastHeartbeatTime int64 `json:"last_heartbeat_time"` // 最后一次接收时间
}

func deepCopyJSON(src interface{}) map[string]interface{} {
	var dest map[string]interface{}
    jsonStr, _ := json.Marshal(src)
    _ = json.Unmarshal(jsonStr, &dest)
    return dest
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
	Timeout           int64              `json:"timeout"`             // 超时时间
	Members           map[string]*Member `json:"members"`             // 所有成员
	Logger            logger.Logger      `json:"-"`
	NoElection        bool               `json:"NoElection"` // 不参与leader选举
}

// ServiceResponseElection 接口服务响应, candidate的投票
// 当接受到别人投票时,自己的状态设置为 follower，并将投票人设置为leader
func (r *Raft) ServiceResponseElection(leader *Leader) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if _, ok := r.Members[leader.LeaderId]; !ok {
		return fmt.Errorf("成员%s在节点中不存在", leader.LeaderId)
	}
	if leader.Term <= r.CurrentTerm {
		return fmt.Errorf("选举者%s的任期%d小于当前任期%d", leader.LeaderId, leader.Term, r.CurrentTerm)
	}
	if r.VotedFor == leader.LeaderId {
		// r.Logger.Debugf("节点已经向%s投票,本次请求不做任何处置", leader.LeaderId)
		// return nil
		return fmt.Errorf("节点已经向%s投票,本次请求不做任何处置", leader.LeaderId)
	}
	if r.VotedFor != "" {
		return fmt.Errorf("节点的选票已经使用,不能给%s投票", leader.LeaderId)
	}
	r.VotedFor = leader.LeaderId
	r.CurrentLeader = leader.LeaderId
	r.CurrentTerm = leader.Term
	r.Logger.Debugf("节点将选票投给%s,设置任期为%d", leader.LeaderId, leader.Term)
	if r.Id != leader.LeaderId {
		r.Role = "follower"
		r.Logger.Debugf("节点的角色设置为follower")
	}
	return nil
}

// ServiceResponseHeartbeat 接口服务响应心跳请求
func (r *Raft) ServiceResponseHeartbeat(leader *Leader) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.LastHeartbeatTime = time.Now().Unix()
	if r.CurrentLeader == leader.LeaderId {
		if r.CurrentTerm != leader.Term {
			r.CurrentTerm = leader.Term
			r.TempTerm = leader.Term
		}
		return nil
	}
	if r.CurrentTerm >= leader.Term {
		return errors.New("leader发生变化,当任期小于原来的任期")
	}
	if r.Id != leader.LeaderId && r.Role != "follower" {
		r.Role = "follower"
	}
	r.CurrentLeader = leader.LeaderId
	r.VotedFor = leader.LeaderId
	r.CurrentTerm = leader.Term
	r.TempTerm = leader.Term
	r.Logger.Debugf("leader和trem发生变化,新leader %s 任期为%d", leader.LeaderId, leader.Term)
	return nil
}

// ServiceResponseSyncMember 接口服务响应同步成员信息
func (r *Raft) ServiceResponseSyncMember(members map[string]*Member) {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	for id, m := range members {
		if _, ok := r.Members[id]; ok {
			r.Members[id] = m
		}
	}
}

// 现所有成员发选举信息
func (r *Raft) sendElectionVotesToAllMember() {
	if r.Role != "candidate" {
		return
	}
	wg := sync.WaitGroup{}
	for _, member := range r.Members {
		wg.Add(1)
		go func(m *Member) {
			defer wg.Done()
			r.Mu.Lock()
			r.Logger.Debugf("election - %s 准备对本节点的投票, 节点状态%s, 获取投票数%d", m.Address, r.Members[m.Id].Status, r.VotedCount)
			r.Mu.Unlock()
			if r.Members[m.Id].Status != "ok" {
				r.Mu.Lock()
				addr := m.Address
				tmpTerm := r.TempTerm
				if r.CurrentTerm > tmpTerm {
					tmpTerm = r.CurrentTerm
				}
				id := r.Id
				r.Mu.Unlock()

				url := "http://" + addr + "/api/v1/election"
				resp, err := grequest.Post(url, &grequest.RequestOptions{
					Data:    Leader{LeaderId: id, Term: tmpTerm + 1},
					Json:    true,
					Timeout: time.Second * 1,
				})
				if err != nil {
					r.Logger.Warnf("election - 拉取%s的选票错误, 错误信息:%s", addr, err.Error())
					return
				}
				if resp.StatusCode() == 201 {
					msg, _ := resp.Text()
					r.Mu.Lock()
					m.Status = "ok"
					r.Mu.Unlock()
					r.Logger.Debugf("election -  拉取%s选取失败,选票投给其他节点 %s", addr, msg)
					return
				}
				if resp.StatusCode() != 200 {
					r.Logger.Warnf("election - %s 拉取选票状态码错误:%s", addr, resp.StatusCode)
					return
				}
				r.Mu.Lock()
				r.Members[id].Status = "ok"
				r.CurrentTerm++
				r.TempTerm++
				r.VotedCount++
				r.Logger.Debugf("election - %s 完成对本节点的投票, 节点状态%s, 获取投票数%d", addr, r.Members[id].Status, r.VotedCount)
				r.Mu.Unlock()
			}
		}(member)
	}
	wg.Wait()
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if r.VotedCount > len(r.Members)/2 {
		r.Role = "leader"
		r.Members[r.Id].Role = "leader"
		r.TempTerm = r.CurrentTerm
	}
}

// 向所有成员发生心跳信息,并向成员发生成员信息
func (r *Raft) sendHeartbeatToAllMember() {
	for _, _member := range r.Members {
		go func(m *Member) {
			r.Mu.Lock()
			add := m.Address
			id := r.Id
			term := r.CurrentTerm
			members := deepCopyJSON(r.Members) 
			r.Mu.Unlock()

			url := "http://" + add + "/api/v1/heartbeat"
			resp, err := grequest.Post(url, &grequest.RequestOptions{
				Data:    Leader{LeaderId: id, Term: term},
				Json:    true,
				Timeout: time.Second * 1,
			})
			if err != nil {
				r.Logger.Warnf("heartbeat - failed to request heartbeat from %s, error message:%s", add, err.Error())
				return
			}
			if resp.StatusCode() == 201 {
				msg, _ := resp.Text()
				m.Status = "ok"
				r.Logger.Debugf("heartbeat - %s 心跳信息处理失败: %s", add, msg)
				return
			}
			if resp.StatusCode() != 200 {
				r.Logger.Warnf("heartbeat - %s 心跳信息发生失败", add)
				return
			}
			r.Members[m.Id].LastHeartbeatTime = time.Now().Unix()
			_, _ = grequest.Post("http://"+add+"/api/v1/sync_member", &grequest.RequestOptions{
				Data:    members,
				Json:    true,
				Timeout: time.Second * 1,
			})
		}(_member)
	}
}

// 后台执行选举任务
func (r *Raft) BackendElection() {
	rand.Seed(time.Now().UnixNano())
	for {
		r.Mu.Lock()
		role := r.Role
		noElection := r.NoElection
		r.Mu.Unlock()

		if role != "candidate" || noElection {
			time.Sleep(time.Second)
			continue
		}
		time.Sleep(time.Duration(rand.Intn(150)+150) * time.Millisecond)
		r.sendElectionVotesToAllMember()
		// d, err := json.Marshal(r)
		// fmt.Println(string(d), err)
	}
}

// 后台有leader发生心跳信息
func (r *Raft) BackendHeatbeat() {
	for {
		r.Mu.Lock()
		role := r.Role
		leader := r.CurrentLeader
		id := r.Id
		r.Mu.Unlock()
		if role == "leader" && leader == id {
			r.Logger.Debugf("send heatbert")
			r.sendHeartbeatToAllMember()
		}
		time.Sleep(time.Second)
	}
}

// leader 联系失败，重新选举
func (r *Raft) BackendReCandidate() {
	for {
		time.Sleep(time.Second * 2)
		r.Mu.Lock()
		role := r.Role
		members := r.Members
		heartbeatTime := r.LastHeartbeatTime
		timeout := r.Timeout
		r.Mu.Unlock()

		if role == "candidate" {
			for _, m := range members {
				if time.Now().Unix()-m.LastHeartbeatTime > r.Timeout {
					r.Mu.Lock()
					for id := range r.Members {
						r.Members[id].Status = ""
					}
					r.Mu.Unlock()
				}
			}
			r.VotedFor = ""
			//r.VotedCount = 0
			r.Logger.Debugf("初始化 - 锁定选票超时, 节点可投新的选票")
		}
		if role == "candidate" || heartbeatTime == 0 {
			continue
		}
		if time.Now().Unix()-heartbeatTime > timeout {
			r.Mu.Lock()
			r.Role = "candidate"
			r.CurrentLeader = ""
			r.VotedCount = 0
			r.VotedFor = ""
			for id := range r.Members {
				r.Members[id].LeaderId = ""
				r.Members[id].Role = ""
				r.Members[id].Status = ""
			}
			r.Mu.Unlock()
			r.Logger.Debugf("初始化 - 心跳超时，节点初始化重新参与选举")
		}
	}

}
