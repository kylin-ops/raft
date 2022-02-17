package tools

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type body struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Info interface{} `json:"info"`
}

func ApiResponse(resp http.ResponseWriter, code int, data, info interface{}) {
	resp.WriteHeader(code)
	resp.Header().Set("statusCode", strconv.Itoa(code))
	d, _ := json.Marshal(body{Code: code, Data: data, Info: info})
	_, _ = resp.Write([]byte(d))
}
