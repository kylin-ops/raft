package raft

import (
	"fmt"
	"time"
)

func (r *Raft)ElectionResponse(leader *Leader)error{
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if r.VotedFor != "" {
		return fmt.Errorf("响应投票请求 - %s的投票请求失败,选票已经投给%s", leader.LeaderId, r.VotedFor)
	}
	r.CurrentLeader = leader.LeaderId
	r.VotedFor = leader.LeaderId
	if r.Id != leader.LeaderId {
		r.Role = "follower"
	}
	return nil
}

func (r *Raft)HeartbeatResponse(body *HeartbeatBody)error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.VotedFor = body.Leader
	if r.Id == body.Leader {
		r.Role = "leader"
	}else{
		r.Role = "follower"
	}
	r.CurrentLeader = body.Leader
	r.Members = body.Members
	r.LastHeartbeatTime = time.Now().Unix()
	r.Logger.Debugf("接收到来自%s的心跳信息", body.Leader)
	return nil
}

func (r *Raft)SetLeaderResponse(id string)error{
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if _, ok := r.Members[id]; !ok {
		return fmt.Errorf("设置leader指定的成员%s不存在", id)
	}
	return nil
}