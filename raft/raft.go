package raft

import (
	"encoding/json"
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

// func deepCopyJSON(src interface{}) map[string]interface{} {
// 	var dest map[string]interface{}
//     jsonStr, _ := json.Marshal(src)
//     _ = json.Unmarshal(jsonStr, &dest)
//     return dest
// }

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
	// if r.CurrentTerm >= leader.Term {
	// 	return errors.New("leader发生变化,当任期小于原来的任期")
	// }
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

func (r *Raft) sendElectionVotes(m *Member) {
	if m.Status == "ok" {
		r.Logger.Debugf("%s节点准备向%s节点发起投票请求 - %s选票已经使用,不需要重复投票,当前已获得选票数%d", r.Id, m.Id, m.Id, r.VotedCount)
		return
	}
	r.Logger.Debugf("%s节点准备向%s发起投票请求 - 当前已获得选票数%d", r.Id, m.Id, r.VotedCount)
	tmpTerm := r.TempTerm
	if r.CurrentTerm > tmpTerm {
		tmpTerm = r.CurrentTerm
	}
	resp, err := grequest.Post("http://"+m.Address+"/api/v1/election", &grequest.RequestOptions{
		Data:    Leader{LeaderId: r.Id, Term: tmpTerm + 1},
		Json:    true,
		Timeout: time.Second * 1,
	})
	if err != nil {
		r.Logger.Warnf("%s节点请求%s节点的选票 - 请求地址:%s, 请求错误, 错误信息:%s", r.Id, m.Id, m.Address, err.Error())
		return
	}
	if resp.StatusCode() == 201 {
		msg, _ := resp.Text()
		r.Mu.Lock()
		r.Members[r.Id].Status = "ok"
		r.Mu.Unlock()
		r.Logger.Debugf("%s节点请求%s节点的选票 - 选票投给其他节点 %s", r.Id, m.Id, msg)
		return
	}
	if resp.StatusCode() != 200 {
		r.Logger.Warnf("%s节点请求%s节点的选票 - 拉取选票状态码错误:%s", r.Id, m.Id, resp.StatusCode)
		return
	}
	r.Mu.Lock()
	r.Members[m.Id].Status = "ok"
	r.CurrentTerm++
	r.TempTerm = r.CurrentTerm
	r.VotedCount++
	r.Logger.Debugf("%s节点请求%s节点的选票 - 完成对本节点的投票, 节点角色%s, 获取投票数%d", r.Id, m.Id, r.Role, r.VotedCount)
	r.Mu.Unlock()

}

// 现所有成员发选举信息
func (r *Raft) sendElectionVotesToAllMember() {
	if r.Role != "candidate" {
		return
	}
	var members map[string]*Member
	r.Mu.Lock()
	data, _ := json.Marshal(r.Members)
	r.Mu.Unlock()
	_ = json.Unmarshal(data, &members)
	wg := sync.WaitGroup{}
	for _, member := range members {
		wg.Add(1)
		go func(m *Member) {
			defer wg.Done()
			r.sendElectionVotes(m)
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
	var members map[string]*Member
	r.Mu.Lock()
	data, _ := json.Marshal(r.Members)
	r.Mu.Unlock()
	_ = json.Unmarshal(data, &members)

	for _, _member := range members {
		go func(m *Member) {
			r.Mu.Lock()
			add := m.Address
			id := r.Id
			term := r.CurrentTerm
			r.Mu.Unlock()

			url := "http://" + add + "/api/v1/heartbeat"
			resp, err := grequest.Post(url, &grequest.RequestOptions{
				Data:    Leader{LeaderId: id, Term: term},
				Json:    true,
				Timeout: time.Second * 1,
			})
			if err != nil {
				r.Logger.Warnf("heartbeat - 向%s发送心跳信息错误, 错误信息:%s", add, err.Error())
				return
			}
			if resp.StatusCode() == 201 {
				msg, _ := resp.Text()
				r.Mu.Lock()
				r.Members[m.Id].Status = "ok"
				r.Mu.Unlock()
				r.Logger.Debugf("heartbeat - %s 心跳信息处理失败: %s", add, msg)
				return
			}
			if resp.StatusCode() != 200 {
				r.Logger.Warnf("heartbeat - %s 心跳信息发生失败", add)
				return
			}
			r.Mu.Lock()
			r.Members[m.Id].LastHeartbeatTime = time.Now().Unix()
			r.Mu.Unlock()
			r.Mu.Lock()
			_, _ = grequest.Post("http://"+add+"/api/v1/sync_member", &grequest.RequestOptions{
				Data:    members,
				Json:    true,
				Timeout: time.Second * 1,
			})
			r.Mu.Unlock()
		}(_member)
	}
}

// 后台执行选举任务
func (r *Raft) BackendElection() {
	for {
		rand.Seed(time.Now().UnixNano())
		if r.Role != "candidate" || r.NoElection {
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
		if (r.Role == "leader" && r.CurrentLeader == r.Id) || len(r.Members)/2 < r.VotedCount {
			r.Logger.Debugf("send heatbert")
			r.sendHeartbeatToAllMember()
		} 
		// else {
		// 	r.Logger.Debugf("heartbeat - 节点%s的角色是%s,当前的leader是%s,选票有%d", r.Id, r.Role, r.CurrentLeader, r.VotedCount)
		// }
		time.Sleep(time.Second)
	}
}

// leader 联系失败，重新选举
func (r *Raft) BackendReCandidate() {
	go func() {
		for {
			time.Sleep(time.Second * 1)
			var members map[string]*Member
			r.Mu.Lock()
			data, _ := json.Marshal(r.Members)
			r.Mu.Unlock()
			_ = json.Unmarshal(data, &members)
			if r.Role == "candidate" {
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
					r.Members[id].Status = ""
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
