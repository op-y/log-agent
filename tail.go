package main

import (
    "log"
    "os"
)

type FileAgent struct {
    Filename     string
    File         *os.File
    FileInfo     os.FileInfo
    LastOffset   int64
    UnchangeTime int
    TsEnabled    bool
    TsPattern    string
    CurrentTs    string
    Tasks        []*AgentTask
}

type AgentTask struct {
    Metric       string
    Tags         string
    CounterType  string
    Step         int64
    Pattern      string
    Method       string
    ValueCnt     int64
    ValueMax     float64
    ValueMin     float64
    ValueAvg     float64
    ValueSum     float64
}

func TailForever(fa *FileAgent, finish <-chan bool) {
    log.Printf("agent for %s launch...", fa.Filename);

    TAIL:
    for {
        select {
        case <-finish:
            break TAIL
        default:
            log.Printf("tail fil %s", fa.Filename)
        }
            
    }

    log.Printf("agent for %s exit...", fa.Filename);
}
