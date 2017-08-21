package main

import (
	"bytes"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	CONFIG_CHECK_INTERVAL = 5

	MAX_UNCHANGED_TIME = 5
)

type Record struct {
	Name   string
	Agent  *FileAgent
	Finish chan bool
}

var config *Config
var configMD5Sum []byte
var records []*Record
var wg sync.WaitGroup

func main() {
	sysCh := make(chan os.Signal, 1)
	signal.Notify(sysCh, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer close(sysCh)

	ticker := time.NewTicker(CONFIG_CHECK_INTERVAL * time.Second)
	defer ticker.Stop()

	// set log format
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// load configuration
	config = LoadConfig()
	if config == nil {
		log.Printf("configuration loading error, please check the config.yaml")
		os.Exit(-1)
	}

	// check configuration md5
	configMD5Sum = CheckConfigMD5()

	DispatchAgent()

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

	wg.Wait()
	log.Printf("log-agent exit...")
}

func DispatchAgent() {
	for _, one := range config.Logs {

		var tasks []*AgentTask
		for _, item := range one.Items {
			task := new(AgentTask)

			task.Metric = item.Metric
			task.Tags = item.Tags
			task.CounterType = item.CounterType
			task.Step = item.Step
			task.Pattern = item.Pattern
			task.Method = item.Method
			task.TsStart = 0
			task.TsEnd = 0
			task.TsUpdate = 0
			task.ValueCnt = 0
			task.ValueMax = 0
			task.ValueMin = 1<<32
			task.ValueSum = 0

			tasks = append(tasks, task)
		}

		agent := new(FileAgent)
		agent.Filename = one.Path
		agent.File = nil
		agent.FileInfo = nil
		agent.LastOffset = 0
		agent.UnchangeTime = 0
		agent.Delimiter = one.Delimiter
		agent.TsEnabled = one.TsEnabled
		agent.TsPattern = one.TsPattern
		agent.Tasks = tasks

		name := one.Name
		ch := make(chan bool, 1)

		record := new(Record)
		record.Name = name
		record.Agent = agent
		record.Finish = ch

		records = append(records, record)
	}

	for _, record := range records {
		wg.Add(1)
		log.Printf("wg: %v", wg)
		go TailForever(record.Agent, record.Finish)
	}
}

func RecallAgent() {
	log.Printf("Before recall, length of records: %d", len(records))

	for _, record := range records {
		record.Finish <- true

		close(record.Finish)
	}

	records = []*Record{}
	log.Printf("After recall, length of records: %d", len(records))
}

func RecheckConfig() {
	newMD5Sum := CheckConfigMD5()
	if !bytes.Equal(configMD5Sum, newMD5Sum) {
		log.Printf("oldMD5Sum %x ----- newMD5Sum %x", configMD5Sum, newMD5Sum)
		cfg := LoadConfig()
		if cfg == nil {
			log.Printf("configuration loading error, please check the config.yaml!")
			return
		}
		config = cfg
		configMD5Sum = newMD5Sum

		RecallAgent()
		DispatchAgent()
	}
}
