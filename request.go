// 发生请求信息
package raft

import (
	"sync"
	"time"

	"github.com/kylin-ops/raft/http/httpclient/grequest"
)

// 发送选举信息
func (r *Raft) requestElection(m *Member) {
	if !(m.Role == "" || m.Role == "error") {
		return
	}
	if r.Role != "candidate" {
		r.Logger.Debugf("节点%s的状态不是candidate,不能发送选票", r.Id)
		return
	}

	addr := "http://" + m.Address + "/api/v1/election"
	resp, err := grequest.Post(addr, &grequest.RequestOptions{
		Data:    Leader{LeaderId: r.Id},
		Json:    true,
		Timeout: time.Second * 1,
	})
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if err != nil {
		r.Members[m.Id].ElectionStatus = "error"
		r.Logger.Warnf("向%s请求选票错误，错误信息:%s", m.Id, err.Error())
		return
	}
	if resp.StatusCode() == 201 {
		r.Members[m.Id].ElectionStatus = "failed"
		msg, _ := resp.Text()
		r.Logger.Warnf("向%s请求选票失败:%s", m.Id, msg)
		return
	}
	if resp.StatusCode() != 200 {
		msg, _ := resp.Text()
		r.Members[m.Id].ElectionStatus = "error"
		r.Logger.Warnf("向%s请求选选票错误，错误信息:%s", m.Id, msg)
	}

	r.VotedCount++
	if r.VotedCount > len(r.Members)/2 {
		r.Role = "leader"
	}
	r.Members[m.Id].ElectionStatus = "ok"
	r.LastHeartbeatTime = time.Now().Unix()
	
	r.Logger.Debugf("向%s请求选票成功,当前选票数%d", m.Id, r.VotedCount)
}

// 发送心跳信息
func (r *Raft) requestHeartbeat(m *Member, heart *HeartbeatBody){
	if r.Role != "leader" {
		r.Logger.Debugf("%s不是leader不能发送心跳信息", r.Id)
		return
	}
	addr := "http://" + m.Address + "/api/v1/heartbeat"
	resp, err := grequest.Post(addr, &grequest.RequestOptions{
		Data:    heart,
		Json:    true,
		Timeout: time.Second * 1,
	})
	if err != nil {
		r.Logger.Warnf("向%s发送心跳错误，错误信息:%s", m.Id, err.Error())
		
		return
	}
	if resp.StatusCode() != 200 {
		msg, _ := resp.Text()
		r.Logger.Warnf("向%s发送心跳失败:%s", m.Id, msg)
		return
	}
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.Members[m.Id].HeartbeatStatus = "online"
	r.Members[m.Id].LastHeartbeatTime = time.Now().Unix()
	r.Members[m.Id].LeaderId = r.CurrentLeader
	role := "follower"
	if m.Id == r.Id {
		role = "leader"
	}
	r.Members[m.Id].Role = role
	r.Logger.Debugf("向%s发送心跳成功", m.Id)
}

// 向所有成员发生选举信息
func(r *Raft) sendElectionToALLMembers(){
	members := r.GetMembers()
	wg := sync.WaitGroup{}
	for _, member := range members {
		if !(member.Role == "" || member.Role == "error") {
			continue
		}
		wg.Add(1)
		go func (m *Member)  {
			defer wg.Done()
			r.requestElection(m)
		}(member)
	}
	wg.Wait()
}

// 向所有成员发生心跳信息
func (r *Raft) sendHeartbeatToAllMembers(){
	members := r.GetMembers()
	wg := sync.WaitGroup{}
	for _, member := range members {
		wg.Add(1)
		go func (m *Member)  {
			defer wg.Done()
			r.requestHeartbeat(m, &HeartbeatBody{Leader: r.Id, Members: members})
		}(member)
	}
	wg.Wait()
}