package controller

import (
	"encoding/json"
	"github.com/kylin-ops/raft/http/httpserver/tools"
	"github.com/kylin-ops/raft/raft"
	"io/ioutil"
	"net/http"
)

func ElectionRequest(resp http.ResponseWriter, req *http.Request) {
	var body raft.Leader
	data, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(data, &body)
	if err := raft.RaftInstance.ServiceResponseElection(&body); err != nil {
		raft.RaftInstance.Logger.Warnf("response election - %s", err.Error())
		tools.ApiResponse(resp, 201, "", err.Error())
		return
	}
	tools.ApiResponse(resp, 200, "", "")
}

func HeartbeatRequest(resp http.ResponseWriter, req *http.Request) {
	var body raft.Leader
	data, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(data, &body)
	if err := raft.RaftInstance.ServiceResponseHeartbeat(&body); err != nil {
		raft.RaftInstance.Logger.Warnf("response heartbeat - %s", err.Error())
		tools.ApiResponse(resp, 201, "", err.Error())
		return
	}
	tools.ApiResponse(resp, 200, "", "")
}

func SyncMemberRequest(resp http.ResponseWriter, req *http.Request) {
	var body map[string]*raft.Member
	data, _ := ioutil.ReadAll(req.Body)
	_ = json.Unmarshal(data, &body)
	raft.RaftInstance.ServiceResponseSyncMember(body)
	tools.ApiResponse(resp, 200, "", "")
}

func GetRaftInfo(resp http.ResponseWriter, req *http.Request){
	tools.ApiResponse(resp, 200, raft.RaftInstance, "")
}