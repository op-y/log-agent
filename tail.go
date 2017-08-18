package main

import (
	"log"
	"os"
	"time"
)

type FileAgent struct {
	Filename     string
	File         *os.File
	FileInfo     os.FileInfo
	LastOffset   int64
	UnchangeTime int
	Delimiter    string
	TsEnabled    bool
	TsPattern    string
	Tasks        []*AgentTask
}

type AgentTask struct {
	Metric      string
	Tags        string
	CounterType string
	Step        int64
	CurrentTs   string
	Pattern     string
	Method      string
	ValueCnt    int64
	ValueMax    float64
	ValueMin    float64
	ValueAvg    float64
	ValueSum    float64
}

func (fa *FileAgent) TimeupAgent() {
	if fa.TsEnabled {
		return
	}
}

func TailForever(fa *FileAgent, finish <-chan bool) {
	log.Printf("agent for %s launch...", fa.Filename)

	ticker := time.NewTicker(1e9)
	defer ticker.Stop()

TAIL:
	for {
		select {
		case <-finish:
			break TAIL
		case <-ticker.C:
			fa.TimeupAgent()
		default:
			time.Sleep(1e9)
		}
	}

	wg.Done()
	log.Printf("wg: %v", wg)
	log.Printf("agent for %s exit...", fa.Filename)
}
