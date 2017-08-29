/*
* tail.go - file agent data structure and funtions to tail file
*
* history
* --------------------
* 2017/8/18, by Ye Zhiqin, create
*
* DESCRIPTION
* This file contains the definition of file agent
* and the functions to tail log file
*/

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

type FileAgent struct {
	Filename      string
	File          *os.File
	FileInfo      os.FileInfo
	LastOffset    int64
	UnchangeTime  int
	Delimiter     string
	TsEnabled     bool
	TsPattern     string
	Tasks         []*AgentTask
}

type AgentTask struct {
	Metric      string
	Tags        string
	CounterType string
	Step        int64
	TsStart     int64
	TsEnd       int64
	TsUpdate    int64
	Pattern     string
	Method      string
	ValueCnt    int64
	ValueMax    float64
	ValueMin    float64
	ValueSum    float64
}

/*
* Update - push and update data after a period passed
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   - ts: timestamp
*   - tsEnabled: is log file timestamp enabled
*
* RETURNS:
*   No paramter
*/
func (task *AgentTask) Update(ts time.Time, tsEnabled bool) {
	var data []*FalconData

	if task.Method == "count" {
		metricCnt := task.Metric + ".cnt"
		point := NewFalconData(metricCnt, config.Falcon.Endpoint, task.ValueCnt, task.CounterType, task.Tags, task.TsEnd, task.Step)
		data = append(data, point)
	}

	if task.Method == "statistic" {
		metricCnt := task.Metric + ".cnt"
		point := NewFalconData(metricCnt, config.Falcon.Endpoint, task.ValueCnt, task.CounterType, task.Tags, task.TsEnd, task.Step)
		data = append(data, point)

		metricMax := task.Metric + ".max"
		point = NewFalconData(metricMax, config.Falcon.Endpoint, task.ValueMax, task.CounterType, task.Tags, task.TsEnd, task.Step)
		data = append(data, point)

		metricMin := task.Metric + ".min"
		if task.ValueMin > task.ValueMax {
			point = NewFalconData(metricMin, config.Falcon.Endpoint, 0, task.CounterType, task.Tags, task.TsEnd, task.Step)
			data = append(data, point)
		} else {
			point = NewFalconData(metricMin, config.Falcon.Endpoint, task.ValueMin, task.CounterType, task.Tags, task.TsEnd, task.Step)
			data = append(data, point)
		}

		metricAvg := task.Metric + ".avg"
		if task.ValueCnt == 0 {
			point = NewFalconData(metricAvg, config.Falcon.Endpoint, 0, task.CounterType, task.Tags, task.TsEnd, task.Step)
			data = append(data, point)
		} else {
			point = NewFalconData(metricAvg, config.Falcon.Endpoint, task.ValueSum/float64(task.ValueCnt), task.CounterType, task.Tags, task.TsEnd, task.Step)
			data = append(data, point)
		}
	}

	log.Printf("falcon point: %v", data)
	response, err := PushData(config.Falcon.Url, data)
	if err != nil {
		log.Printf("push data to falcon ERROR: %v", err)
	}
	log.Printf("push data to falcon succeed: %s", string(response))

	// update value
	task.ValueCnt = 0
	task.ValueMax = 0
	task.ValueMin = 1 << 32
	task.ValueSum = 0

	//update timestamp
	if tsEnabled {
		minute := ts.Format("200601021504")
		start, err := time.ParseInLocation("20060102150405", minute+"00", ts.Location())
		if err != nil {
			log.Printf("timestamp setting ERROR: %v", err)
		}
		tsStart := start.Unix()

		end, err := time.ParseInLocation("20060102150405", minute+"59", ts.Location())
		if err != nil {
			log.Printf("timestamp setting ERROR: %v", err)
		}
		tsEnd := end.Unix()
		task.TsStart = tsStart
		task.TsEnd = tsEnd
		task.TsUpdate = ts.Unix()
	} else {
		task.TsStart += task.Step
		task.TsEnd += task.Step
		task.TsUpdate = ts.Unix()
	}
}

/*
* MatchLine - process each line of log
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   - line: a line of log file
*
* RETURNS:
*   No paramter
*/
func (fa *FileAgent) MatchLine(line []byte) {
	if fa.TsEnabled {

		isTsMatched, ts, err := MatchTs(line, fa.TsPattern)
		if err != nil || !isTsMatched {
			log.Printf("Timestamp NOT matched!")
			return
		}

		for _, task := range fa.Tasks {
			//push data and update task when the time between this line and start timestamp is longer than a step
			if ts.Unix() > task.TsEnd || ts.Unix() < task.TsStart {
				log.Printf("Timestamp updated!")
				task.Update(ts, true)
			}

			if task.Method == "count" {
				log.Printf("Task count!")
				isKeywordMatched, err := MatchKeyword(line, task.Pattern)
				if err != nil || !isKeywordMatched {
					return
				}
				task.ValueCnt += 1
				task.TsUpdate = ts.Unix()
			}

			if task.Method == "statistic" {
				log.Printf("Task statistic: %s", task.Pattern)

				isCostMatched, cost, err := MatchCost(line, task.Pattern)
				if err != nil || !isCostMatched {
					log.Printf("Cost NOT matched!")
					return
				}
				log.Printf("Bravo: %f !", cost)

				task.ValueCnt += 1
				if task.ValueMax < cost {
					task.ValueMax = cost
				}
				if task.ValueMin > cost {
					task.ValueMin = cost
				}
				task.ValueSum += cost
				task.TsUpdate = ts.Unix()
			}

			log.Printf("Task: %v", task)
		}
	} else {
		for _, task := range fa.Tasks {
			if task.Method == "count" {
				isKeywordMatched, err := MatchKeyword(line, task.Pattern)
				if err != nil || !isKeywordMatched {
					return
				}
				task.ValueCnt += 1
			}

			if task.Method == "statistic" {
				isCostMatched, cost, err := MatchCost(line, task.Pattern)
				if err != nil || !isCostMatched {
					return
				}
				task.ValueCnt += 1
				if task.ValueMax < cost {
					task.ValueMax = cost
				}
				if task.ValueMin > cost {
					task.ValueMin = cost
				}
				task.ValueSum += cost
			}
		}
	}
}

/*
* Timeup - the process after a period passed
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   No paramter
*/
func (fa *FileAgent) Timeup() {
	ts := time.Now()

	for _, task := range fa.Tasks {
		if fa.TsEnabled {
			if ts.Unix()-task.TsUpdate >= task.Step {
				//push data and update task when the time between now and last update timestamp is longer than a step
				task.Update(ts, true)
			}
		} else {
			if ts.Unix()-task.TsStart >= task.Step {
				//push data and update task when the time between now and start timestamp is longer than a step
				task.Update(ts, false)
			}
		}
	}
}

/*
* IsChanged - check the change of log file
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   - true: if change
*   - false: if not change
*/
func (fa *FileAgent) IsChanged() bool {
	lastMode := fa.FileInfo.Mode()
	lastSize := fa.FileInfo.Size()
	lastModTime := fa.FileInfo.ModTime().Unix()

	fileinfo, err := fa.File.Stat()
	if err != nil {
		log.Printf("file %s stat ERROR: %v", err)
		fa.UnchangeTime += 1
		return false
	}

	thisMode := fileinfo.Mode()
	thisSize := fileinfo.Size()
	thisModTime := fileinfo.ModTime().Unix()
	thisTailable := fileinfo.Mode().IsRegular()

	if lastMode == thisMode &&
		(!thisTailable || lastSize == thisSize) &&
		lastModTime == thisModTime {

		fa.UnchangeTime += 1
		return false
	}

	log.Printf("file %s is changed", fa.Filename)
	fa.UnchangeTime = 0
	fa.FileInfo = fileinfo
	return true
}

/*
* Recheck - recheck the file of file agent
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   No return value
*/
func (fa *FileAgent) Recheck() error {
	filename := fa.Filename

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("file %s opening ERROR: %v", fa.Filename, err)
		fa.UnchangeTime = 0
		return err
	}

	fileinfo, err := file.Stat()
	if err != nil {
		log.Printf("file %s stat ERROR: %v", fa.Filename, err)
		fa.UnchangeTime = 0
		return err
	}

	isNewFile := os.SameFile(fa.FileInfo, fileinfo)
	if !isNewFile {
		log.Printf("file %s recheck, it is a new file", fa.Filename)
		if fa.File != nil {
			if err := fa.File.Close(); err != nil {
				log.Printf("old file closing ERROR: %v", err)
			}
		}

		fa.File = file
		fa.FileInfo = fileinfo
		fa.LastOffset = 0
		fa.UnchangeTime = 0

		offset, err := fa.File.Seek(fa.FileInfo.Size(), os.SEEK_SET)
		log.Printf("File(seek) %s offset %d", fa.Filename, offset)
		if err != nil {
			log.Printf("file seeking ERROR: %v", err)
		}
		fa.LastOffset += fa.FileInfo.Size()
		log.Printf("file %s, size: %d --- offset: %d", fa.Filename, fa.FileInfo.Size(), fa.LastOffset)

		now := time.Now()
		minute := now.Format("200601021504")

		tsNow := now.Unix()
		tsStart := tsNow
		tsEnd := tsNow

		start, err := time.ParseInLocation("20060102150405", minute+"00", now.Location())
		if err != nil {
			log.Printf("timestamp setting ERROR: %v", err)
		} else {
			tsStart = start.Unix()
		}

		end, err := time.ParseInLocation("20060102150405", minute+"59", now.Location())
		if err != nil {
			log.Printf("timestamp setting ERROR: %v", err)
		} else {
			tsEnd = end.Unix()
		}

		for _, task := range fa.Tasks {
			task.TsStart = tsStart
			task.TsEnd = tsEnd
			task.TsUpdate = tsNow
			task.ValueCnt = 0
			task.ValueMax = 0
			task.ValueMin = 1 << 32
			task.ValueSum = 0

			log.Printf("TS: %d %d %d", task.TsStart, task.TsEnd, task.TsUpdate)
		}

		return nil
	} else {
		fa.UnchangeTime = 0
		return nil
	}
}

/*
* ReadRemainder - reading new bytes of log file
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   No return value
*/
func (fa *FileAgent) ReadRemainder() {
	tailable := fa.FileInfo.Mode().IsRegular()
	size := fa.FileInfo.Size()

	if tailable && fa.LastOffset > size {
		log.Printf("file %s be truncated", fa.Filename)
		offset, err := fa.File.Seek(0, os.SEEK_SET)
		log.Printf("File(seek) %s offset %d", fa.Filename, offset)
		if err != nil {
			log.Printf("file %s seek ERROR: %v", fa.Filename, err)
		}
		if offset != 0 {
			log.Printf("offset is not equal 0")
		}
		fa.LastOffset = 0
		log.Printf("file %s, size: %d --- offset: %d", fa.Filename, fa.FileInfo.Size(), fa.LastOffset)

		return
	}

	bufsize := size - fa.LastOffset
	if bufsize == 0 {
		return
	}
	data := make([]byte, bufsize)
	readsize, err := fa.File.Read(data)

	if err != nil && err != io.EOF {
		log.Printf("file %s read ERROR: %v", err)
		return
	}
	if readsize == 0 {
		log.Printf("file %s read 0 data", fa.Filename)
		return
	}

	if fa.Delimiter == "" {
		fa.Delimiter = "\n"
	}
	sep := []byte(fa.Delimiter)
	lines := bytes.SplitAfter(data, sep)
	length := len(lines)

	for idx, line := range lines {
		if idx == length-1 {
			backsize := len(line)
			movesize := readsize - backsize
			log.Printf("backsize: %d, movesize: %d", backsize, movesize)

			offset, err := fa.File.Seek(-int64(backsize), os.SEEK_CUR)
			if err != nil {
				log.Printf("file %s seek ERROR: %v", fa.Filename, err)
			}
			log.Printf("File(seek) %s offset %d", fa.Filename, offset)
			fa.LastOffset += int64(movesize)
			log.Printf("file %s, size: %d --- offset: %d", fa.Filename, fa.FileInfo.Size(), fa.LastOffset)
			break
		}

		fmt.Printf("line %d: %s", idx, string(line))
		fa.MatchLine(line)
	}
	return
}

/*
* TryReading - reading log file
*
* PARAMS:
*   - fa: file agent
*
* RETURNS:
*   No return value
*/
func TryReading(fa *FileAgent) {
	if fa.File == nil {
		log.Printf("file %s is nil", fa.Filename)
		if err := fa.Recheck(); err != nil {
			log.Printf("file recheck ERROR: %v", err)
		}
		return
	}

	if !fa.IsChanged() {
		//log.Printf("file %s is not changed", fa.Filename)
		if fa.UnchangeTime >= MAX_UNCHANGED_TIME {
			fa.Recheck()
		}
		return
	}

	fa.ReadRemainder()
}

/*
* TailForever - tail log file in a loop
*
* PARAMS:
*   - fa: file agent
*   - finish: a channel to receiver stop signal
*
* RETURNS:
*   No return value
*/
func TailForever(fa *FileAgent, finish <-chan bool) {
	log.Printf("agent for %s launch...", fa.Filename)

	ticker := time.NewTicker(1e9)
	defer ticker.Stop()

TAIL:
	for {
		select {
		case <-finish:
			if fa.File != nil {
				if err := fa.File.Close(); err != nil {
					log.Printf("file closing ERROR: %v", err)
				}
			}
			break TAIL
		case <-ticker.C:
			fa.Timeup()
		default:
    		TryReading(fa)
            time.Sleep(time.Millisecond * 100)
		}
	}

	wg.Done()
	log.Printf("wg: %v", wg)
	log.Printf("agent for %s exit...", fa.Filename)
}
