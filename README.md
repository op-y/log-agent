# log-agent (debugging)
log-agent 跟踪普通可读文件，正则匹配给定的pattern，将匹配结果当做监控数据推送到open falcon系统中

## 文件说明:
* agent.go    -- agent控制和调度
* config.go   -- 配置读取/加载/更新
* config.yaml -- 配置文件
* control     -- 控制脚本
* falcon.go   -- open falcon
* re.go       -- 匹配pattern
* tail.go     -- 文件跟踪

## 使用方法
1. git clone https://github.com/op-y/log-agent.git
2. cd log-agent
3. go build -o log-agent
4. 根据实际情况修改config.yaml配置
5. ./control start
