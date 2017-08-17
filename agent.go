package main

import (
    "bytes"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"
)

const (
    CONFIG_CHECK_INTERVAL = 10

    MAX_UNCHANGED_TIME = 5

    POS_START   = 0
    POS_CURRENT = 1
    POS_END     = 2
)

var config Config
var configMD5Sum []byte
var agents []*FileAgent

func main() {
    sysCh := make(chan os.Signal, 1)
    signal.Notify(sysCh, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
   
    ticker := time.NewTicker(CONFIG_CHECK_INTERVAL * time.Second)
    defer ticker.Stop()

    // set log format
    log.SetFlags(log.LstdFlags | log.Lshortfile)

    // load configuration
    LoadConfig()
    // verify configuration
    VerifyConfig()
    // check configuration md5 
    configMD5Sum = CheckConfigMD5()


    MAIN:
    for {
        select {
        case <-sysCh:
            log.Printf("system signal: %v", sysCh)
            RecallAgent()
            break MAIN
        case <-ticker.C:
            RecheckConfig()
        }   
    } 

    log.Printf("log-agent exit...")
}

func DispatchAgent() {
}

func RecallAgent() {
}

func RecheckConfig() {
    newMD5Sum := CheckConfigMD5()
    log.Printf("oldMD5Sum %x ----- newMD5Sum %x", configMD5Sum, newMD5Sum)
    if ! bytes.Equal(configMD5Sum, newMD5Sum) {
        RecallAgent()
        DispatchAgent()
        configMD5Sum = newMD5Sum 
    }
}
