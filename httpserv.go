// http服务
package raft

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/kylin-ops/raft/http/httpserver/tools"
)

func (r *Raft) httpServer() {
	http.HandleFunc("/api/v1/election", r.electionRequest)
	http.HandleFunc("/api/v1/heartbeat", r.heartbeatRequest)
	http.HandleFunc("/api/v1/get_info", r.getRaftInfo)
	if err := http.ListenAndServe(r.Address, nil); err != nil {
		log.Fatalln(err.Error())
	}
}

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

func (r *Raft) getRaftInfo(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("content-type", "application/json")
	r.Mu.Lock()
	tools.ApiResponse(resp, 200, r, "")
	r.Mu.Unlock()
}
