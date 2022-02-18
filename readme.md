# 1 功能
根据raft协议进行多个节点运行是选取leader，有leader向所有成员发生心跳信息

- 核心配置： 参考example/main.go 配置成员并启动服务
- 重要配置：
    NoElection       
	DefaultLeader     string             `json:"default_leader"`      //
	HealthCheckType   string			 `json:"health_check_type"`   
}