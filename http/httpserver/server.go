package httpserver

import (
	"fmt"
	"github.com/kylin-ops/raft/http/httpserver/controller"
	"github.com/kylin-ops/raft/logger"
	"log"
	"net/http"
)

// var _logger logger.Logger = &logger.Log{}

// func accessLog(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		fn(w, r)
// 		code, _ := strconv.Atoi(w.Header().Get("statusCode"))
// 		l := fmt.Sprintf("%-7s %-5d %-22s %s", r.Method, code, r.RemoteAddr, r.URL.RequestURI())
// 		if code < 400 && code > 199 {
// 			_logger.Infof(l)
// 		} else {
// 			_logger.Errorf(l)
// 		}
// 		//_logger.Infof(l)
// 	}
// }

func StartHttpServer(Host string, port int, logg logger.Logger) {
	// if logg != nil {
	// 	_logger = logg
	// }
	addr := fmt.Sprintf("%s:%d", Host, port)
	http.HandleFunc("/api/v1/election", controller.ElectionRequest)
	http.HandleFunc("/api/v1/heartbeat", controller.HeartbeatRequest)
	http.HandleFunc("/api/v1/sync_member", controller.SyncMemberRequest)
	http.HandleFunc("/api/v1/get_info", controller.GetRaftInfo)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalln(err.Error())
	}
}
