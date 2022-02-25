// http服务
package raft

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/kylin-ops/raft/http/httpserver/tools"
)

var (
	UriElection      = "/api/v1/election"
	UriHeartbeat     = "/api/v1/heartbeat"
	UriGetInfo       = "/api/v1/get_info"
	UriAddMember     = "/api/v1/add_member"
	UriDelMember     = "/api/v1/del_member"
	UriAddMemberExec = "/api/v1/add_member/exec"
	UriDelMemberExec = "/api/v1/del_member/exec"
)

func (r *Raft) httpServer() {
	http.HandleFunc(UriElection, r.electionRequest)
	http.HandleFunc(UriHeartbeat, r.heartbeatRequest)
	http.HandleFunc(UriGetInfo, r.getRaftInfo)
	http.HandleFunc(UriAddMember, r.addMember)
	http.HandleFunc(UriAddMemberExec, r.addMemberExec)
	http.HandleFunc(UriDelMemberExec, r.delMemberExec)
	http.HandleFunc(UriDelMember, r.delMember)
	if err := http.ListenAndServe(r.Address, nil); err != nil {
		log.Fatalln(err.Error())
	}
}

// 集群leader选举
func (r *Raft) electionRequest(resp http.ResponseWriter, req *http.Request) {
	var body Leader
	data, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(data, &body)
	if err := r.ElectionResponse(&body); err != nil {
		r.Logger.Warnf("response election - %s", err.Error())
		tools.ApiResponse(resp, 201, "", err.Error())
		return
	}
	tools.ApiResponse(resp, 200, "", "")
}

// 集群心跳
func (r *Raft) heartbeatRequest(resp http.ResponseWriter, req *http.Request) {
	var body HeartbeatBody
	data, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(data, &body)
	if err := r.HeartbeatResponse(&body); err != nil {
		r.Logger.Warnf("response heartbeat - %s", err.Error())
		tools.ApiResponse(resp, 201, "", err.Error())
		return
	}
	tools.ApiResponse(resp, 200, "", "")
}

// 获取集群信息
func (r *Raft) getRaftInfo(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("content-type", "application/json")
	r.Mu.Lock()
	tools.ApiResponse(resp, 200, r, "")
	r.Mu.Unlock()
}

// 添加集群成员，请求转发到leader
func (r *Raft) addMember(resp http.ResponseWriter, req *http.Request) {
	var body map[string]string
	data, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(data, &body)
	resp.Header().Set("content-type", "application/json")
	if err := r.MemberForwardLeaderResponse("add", body["id"], body["address"]); err != nil {
		tools.ApiResponse(resp, 299, "", err.Error())
		return
	}
	tools.ApiResponse(resp, 200, "", "")
}

// 删除集群成员，请求转发到leader
func (r *Raft) delMember(resp http.ResponseWriter, req *http.Request) {
	var body map[string]string
	data, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(data, &body)
	resp.Header().Set("content-type", "application/json")
	if err := r.MemberForwardLeaderResponse("del", body["id"], body["address"]); err != nil {
		tools.ApiResponse(resp, 299, "", err.Error())
		return
	}
	tools.ApiResponse(resp, 200, "", "")
}

// leader执行添加集群成员
func (r *Raft) addMemberExec(resp http.ResponseWriter, req *http.Request) {
	var body map[string]string
	data, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(data, &body)
	resp.Header().Set("content-type", "application/json")
	if err := r.MemberResponse("add", body["id"], body["address"]); err != nil {
		tools.ApiResponse(resp, 299, "", err.Error())
		return
	}
	tools.ApiResponse(resp, 200, "", "")
}

// leader执行删除集群成员
func (r *Raft) delMemberExec(resp http.ResponseWriter, req *http.Request) {
	var body map[string]string
	data, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(data, &body)
	resp.Header().Set("content-type", "application/json")
	if err := r.MemberResponse("del", body["id"], body["address"]); err != nil {
		tools.ApiResponse(resp, 299, "", err.Error())
		return
	}
	tools.ApiResponse(resp, 200, "", "")
}
