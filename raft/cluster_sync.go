package raft

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/kylin-ops/raft/http/httpclient/grequest"
)

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
	r.Members = members
	// for id, m := range members {
	// 	if _, ok := r.Members[id]; ok {
	// 		r.Members[id] = m
	// 	}
	// }
}

func (r *Raft) sendElectionVotes(m *Member) {
	if m.ElectionStatus == "ok" {
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
		r.Members[r.Id].ElectionStatus = "ok"
		r.Mu.Unlock()
		r.Logger.Debugf("%s节点请求%s节点的选票 - 选票投给其他节点 %s", r.Id, m.Id, msg)
		return
	}
	if resp.StatusCode() != 200 {
		r.Logger.Warnf("%s节点请求%s节点的选票 - 拉取选票状态码错误:%s", r.Id, m.Id, resp.StatusCode)
		return
	}
	r.Mu.Lock()
	r.Members[m.Id].ElectionStatus = "ok"
	r.Members[m.Id].HeartbeatStatus = "ok"
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

// 发送心跳信息
func (r *Raft) sendHeartbeat(m *Member) error {
	url := "http://" + m.Address + "/api/v1/heartbeat"
	resp, err := grequest.Post(url, &grequest.RequestOptions{
		Data:    Leader{LeaderId: r.Id, Term: r.CurrentTerm},
		Json:    true,
		Timeout: time.Second * 1,
	})

	r.Mu.Lock()
	r.Members[m.Id].Role = "follower"
	if m.Id == r.Id {
		r.Members[m.Id].Role = "leader"
	}
	r.Members[m.Id].LeaderId = r.Id
	r.Mu.Unlock()

	if err != nil {
		// r.Logger.Warnf("heartbeat - 向%s发送心跳信息错误, 错误信息:%s", m.Address, err.Error())
		if time.Now().Unix()-r.Members[m.Id].LastHeartbeatTime > 5 {
			r.Members[m.Id].HeartbeatStatus = "error"
		}
		return err
	}

	if resp.StatusCode() == 201 {
		msg, _ := resp.Text()
		r.Mu.Lock()
		r.Members[m.Id].ElectionStatus = "ok"
		if time.Now().Unix()-r.Members[m.Id].LastHeartbeatTime > 5 {
			r.Members[m.Id].HeartbeatStatus = "failed"
		}
		r.Mu.Unlock()
		// r.Logger.Debugf("heartbeat - %s 心跳信息处理失败: %s", m.Address, msg)
		return fmt.Errorf("%s 心跳信息处理失败,返回信息:%s", m.Address, msg)
	}
	if resp.StatusCode() != 200 {
		// r.Logger.Warnf("heartbeat - %s 心跳信息发生失败", m.Address)
		r.Mu.Lock()
		if time.Now().Unix()-r.Members[m.Id].LastHeartbeatTime > 5 {
			r.Members[m.Id].HeartbeatStatus = "error"
		}
		r.Mu.Unlock()
		return fmt.Errorf("%s 心跳信息发生失败,返回code:%d", m.Address, resp.StatusCode())
	}
	r.Mu.Lock()
	if _, ok := r.Members[m.Id]; ok {
		//t := time.Now().Unix()
		r.Members[m.Id].ElectionStatus = "ok"
		r.Members[m.Id].HeartbeatStatus = "ok"
		if time.Now().Unix() > r.Members[m.Id].LastHeartbeatTime {
			r.Members[m.Id].LastHeartbeatTime = time.Now().Unix()
		}

		// r.Logger.Infof("heartbeat - 更新%s的hearbeat时间%d", m.Id, t)
	}
	r.Mu.Unlock()
	return nil
}

// 发送同步成员信息
func (r *Raft) syncMember(m *Member) error {
	var members map[string]*Member
	r.Mu.Lock()
	data, _ := json.Marshal(r.Members)
	r.Mu.Unlock()
	_ = json.Unmarshal(data, &members)
	fmt.Println(string(data))
	_, err := grequest.Post("http://"+m.Address+"/api/v1/sync_member", &grequest.RequestOptions{
		Data:    members,
		Json:    true,
		Timeout: time.Second * 1,
	})
	return err
}

// 向所有成员发生心跳信息,并向成员发生成员信息
func (r *Raft) sendHeartbeatToAllMember() {
	var members map[string]*Member
	r.Mu.Lock()
	data, _ := json.Marshal(r.Members)
	r.Mu.Unlock()
	_ = json.Unmarshal(data, &members)
	wg := sync.WaitGroup{}
	for _, _member := range members {
		wg.Add(1)
		go func(m *Member) {
			defer wg.Done()
			if err := r.sendHeartbeat(m); err != nil {
				r.Logger.Warnf("heartbeat - %s", err.Error())
				return
			}
			r.syncMember(m)
		}(_member)
	}
	wg.Wait()
}
