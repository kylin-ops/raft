// 响应服务请求
package raft

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/kylin-ops/raft/http/httpclient/grequest"
)

func (r *Raft) ElectionResponse(leader *Leader) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if _, ok := r.Members[leader.LeaderId]; !ok {
		return fmt.Errorf("响应投票请求 - %s不能参与选举，不在成员列表中",leader.LeaderId)
	}

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

func (r *Raft) HeartbeatResponse(body *HeartbeatBody) error {
	err := r.HealthChecker.Do()
	if err != nil {
		d, _ := json.Marshal(r.HealthChecker)
		r.Logger.Warnf("心跳check错误,执行信息:%s 错误信息:%s", string(d), err.Error())
	}

	r.Mu.Lock()
	defer r.Mu.Unlock()
	if r.DefaultLeader == r.Id && r.Id != body.Leader {
		return fmt.Errorf("本节点设置默认leader与心跳leader不一致")
	}
	r.VotedFor = body.Leader
	if r.Id == body.Leader {
		r.Role = "leader"
	} else {
		r.Role = "follower"
	}
	r.CurrentLeader = body.Leader
	if err == nil {
		r.Members = body.Members
		r.LastHeartbeatTime = time.Now().Unix()
		r.Logger.Debugf("接收到来自%s的心跳信息", body.Leader)
	}
	return nil
}

func (r *Raft) SetLeaderResponse(id string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if _, ok := r.Members[id]; !ok {
		return fmt.Errorf("设置leader指定的成员%s不存在", id)
	}
	return nil
}

// 接收添加或删除member请求的信息并转发给leader
func (r *Raft) MemberForwardLeaderResponse(action, id, address string) error {
	var uri string
	r.Mu.Lock()
	m, ok := r.Members[r.CurrentLeader]
	if !ok {
		r.Mu.Unlock()
		return errors.New("集群中没有leader")
	}
	addr := m.Address
	r.Mu.Unlock()
	switch action{
	case "add":
		uri = UriAddMember
	case "del":
		uri = UriDelMember
	default:
		return fmt.Errorf("不支持%s操作", action)
	}
	url := "http://" + addr + uri
	resp, err := grequest.Post(url, &grequest.RequestOptions{
		Json:    true,
		Data:    map[string]string{"id": id, "address": address},
		Timeout: time.Second,
	})
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		msg, _ := resp.Text()
		return errors.New(msg)
	}
	return nil
}

// 添加或删除member
func (r *Raft) MemberResponse(action, id, address string) error {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	if r.Role != "leader" {
		return errors.New("本节点不是leader不能执行添加成员")
	}
	switch action {
	case "add":
		if _, ok := r.Members[id]; ok {
			return fmt.Errorf("成员%s已经存在", id)
		}
		r.Members[id] = &Member{Id: id, Address: address, LeaderId: r.CurrentLeader, Role: "follower"}
	case "del":
		if _, ok := r.Members[id]; !ok {
			return fmt.Errorf("成员%s不存在", id)
		}
		delete(r.Members, id)
	default:
		return fmt.Errorf("不支持%s操作", action)
	}
	return nil
}
