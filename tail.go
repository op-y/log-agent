/*
* tail.go - file agent data structure and funtions to tail file
*
* history
* --------------------
* 2017/8/18, by Ye Zhiqin, create
* 2017/9/30, by Ye Zhiqin, modify
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
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileAgent struct {
	Filename       string
	File           *os.File
	FileInfo       os.FileInfo
	LastOffset     int64
	UnchangeTime   int
	Delimiter      string
	TsEnabled      bool
	TsPattern      string
	InotifyEnabled bool
	Tasks          []*AgentTask
}

type AgentTask struct {
	Metric      string
	Tags        string
	CounterType string
	Step        int64
	Pattern     string
	Method      string
	TsStart     int64
	TsEnd       int64
	TsUpdate    int64
	ValueCnt    int64
	ValueMax    float64
	ValueMin    float64
	ValueSum    float64
}

/*
* Report - push and update data after a period passed
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
func (task *AgentTask) Report(ts time.Time, timeup bool) {
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
		log.Printf("push data to falcon FAIL: %v", err)
	}
	log.Printf("push data to falcon succeed: %s", string(response))

	// update value
	task.ValueCnt = 0
	task.ValueMax = 0
	task.ValueMin = 1 << 32
	task.ValueSum = 0

	//update timestamp
	if timeup {
		task.TsStart += task.Step
		task.TsEnd += task.Step
		task.TsUpdate = task.TsEnd - 1
	} else {
		task.TsStart += task.Step
		task.TsEnd += task.Step
		task.TsUpdate = ts.Unix()
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
				//push data and update task when the time between now and last update time is longer than a step
				task.Report(ts, true)
			}
		} else {
			if ts.Unix()-task.TsStart >= task.Step {
				//push data and update task when the time between now and start time is longer than a step
				task.Report(ts, true)
			}
		}
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
			return
		}

		for _, task := range fa.Tasks {
			//push data and update task when the timestamp is not in current period
			if ts.Unix() > task.TsEnd || ts.Unix() < task.TsStart {
				log.Printf("timestamp updated!")
				task.Report(ts, false)
			}

			if task.Method == "count" {
				isKeywordMatched, err := MatchKeyword(line, task.Pattern)
				if err != nil || !isKeywordMatched {
					continue
				}
				task.ValueCnt += 1
				task.TsUpdate = ts.Unix()
			}

			if task.Method == "statistic" {
				isCostMatched, cost, err := MatchCost(line, task.Pattern)
				if err != nil || !isCostMatched {
					continue
				}

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
		}
	} else {
		for _, task := range fa.Tasks {
			if task.Method == "count" {
				isKeywordMatched, err := MatchKeyword(line, task.Pattern)
				if err != nil || !isKeywordMatched {
					continue
				}
				task.ValueCnt += 1
			}

			if task.Method == "statistic" {
				isCostMatched, cost, err := MatchCost(line, task.Pattern)
				if err != nil || !isCostMatched {
					continue
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
* ReadRemainder - reading new bytes of log file
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   nil: succeed
*   error: fail
 */
func (fa *FileAgent) ReadRemainder() error {
	tailable := fa.FileInfo.Mode().IsRegular()
	size := fa.FileInfo.Size()

	// LastOffset less then new size. Maybe the file has been truncated.
	if tailable && fa.LastOffset > size {
		// seek the cursor to the header of new file
		offset, err := fa.File.Seek(0, os.SEEK_SET)
		if err != nil {
			log.Printf("file %s seek FAIL: %v", fa.Filename, err)
			return err
		}
		if offset != 0 {
			log.Printf("offset is not equal 0")
		}
		fa.LastOffset = 0

		return nil
	}

	bufsize := size - fa.LastOffset
	if bufsize == 0 {
		return nil
	}
	data := make([]byte, bufsize)
	readsize, err := fa.File.Read(data)

	if err != nil && err != io.EOF {
		log.Printf("file %s read FAIL: %v", err)
		return err
	}
	if readsize == 0 {
		log.Printf("file %s read 0 data", fa.Filename)
		return nil
	}

	if fa.Delimiter == "" {
		fa.Delimiter = "\n"
	}
	sep := []byte(fa.Delimiter)
	lines := bytes.SplitAfter(data, sep)
	length := len(lines)

	for idx, line := range lines {
		// just process entire line with the delimiter
		if idx == length-1 {
			backsize := len(line)
			movesize := readsize - backsize

			_, err := fa.File.Seek(-int64(backsize), os.SEEK_CUR)
			if err != nil {
				log.Printf("seek file %s FAIL: %v", fa.Filename, err)
				return err
			}
			fa.LastOffset += int64(movesize)

			break
		}
		fa.MatchLine(line)
	}
	return nil
}

/*
* TailWithCheck - tail log file in a loop
*
* PARAMS:
*   - fa: file agent
*   - finish: a channel to receiver stop signal
*
* RETURNS:
*   No return value
 */
func TailWithCheck(fa *FileAgent, finish <-chan bool) {
	log.Printf("agent for %s is launching...", fa.Filename)

	// create one second ticker
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

TAIL:
	for {
		select {
		case <-finish:
			if fa.File != nil {
				if err := fa.File.Close(); err != nil {
					log.Printf("file closing FAIL: %v", err)
				}
			}
			break TAIL
		case <-ticker.C:
			fa.Timeup()
		default:
			fa.TryReading()
			time.Sleep(time.Millisecond * 100)
		}
	}

	wg.Done()
	log.Printf("wg: %v", wg)
	log.Printf("agent for %s is exiting...", fa.Filename)
}

/*
* TryReading - reading log file
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   No return value
 */
func (fa *FileAgent) TryReading() {
	if fa.File == nil {
		log.Printf("file %s is nil", fa.Filename)
		if err := fa.FileRecheck(); err != nil {
			log.Printf("file recheck FAIL: %v", err)
		}
		return
	}

	if !fa.IsChanged() {
		if fa.UnchangeTime >= MAX_UNCHANGED_TIME {
			fa.FileRecheck()
		}
		return
	}

	fa.ReadRemainder()
}

/*
* TailWithInotify - trace log file
*
* PARAMS:
*   - fa: file agent
*   - finish: a channel to receiver stop signal
*
* RETURNS:
*   No return value
 */
func TailWithInotify(fa *FileAgent, chanFinish <-chan bool) {
	fmt.Printf("agent for %s is starting...\n", fa.Filename)

	// create one second ticker
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	// craete file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("watcher creating FAIL: %s", err.Error())
		wg.Done()
		log.Printf("agent for %s is exiting...", fa.Filename)
		return
	}
	defer watcher.Close()

	// add parent directory to watcher
	dir := path.Dir(fa.Filename)
	if err := watcher.Add(dir); err != nil {
		log.Printf("add %s to watcher FAIL: %s", dir, err.Error())
		wg.Done()
		log.Printf("agent for %s is exiting...", fa.Filename)
		return
	}

	// open file and initialize file agent
	if err := fa.FileOpen(); err != nil {
		log.Printf("file $s open FAIL when agent initializing: %s", fa.Filename, err.Error())
	}

	// trace file in this loop
TRACE:
	for {
		select {
		case <-chanFinish:
			if err := watcher.Remove(dir); err != nil {
				log.Printf("watcher file removing FAIL: %s", err.Error())
			}
			if fa.File != nil {
				if err := fa.File.Close(); err != nil {
					log.Printf("file closing FAIL: %s", err.Error())
				}
			}
			break TRACE
		case event := <-watcher.Events:
			if event.Name == fa.Filename {
				// WRITE event
				if 2 == event.Op {
					if err := fa.FileInfoUpdate(); err != nil {
						log.Printf("file %s stat FAIL", fa.Filename)
						if fa.UnchangeTime > MAX_UNCHANGED_TIME {
							fa.FileReopen()
						}
						continue
					}

					if err := fa.ReadRemainder(); err != nil {
						log.Printf("file %s reading FAIL, recheck it", fa.Filename)
						fa.FileReopen()
					}
				}
				// CREATE event
				if 1 == event.Op {
					fmt.Printf("fa %s, watch %s receive event CREATE\n", fa.Filename, event.Name)
					fa.FileReopen()
				}
				// REMOVE/RENAME event
				if 4 == event.Op || 8 == event.Op {
					fmt.Printf("fa %s, watch %s receive event REMOVE|RENAME\n", fa.Filename, event.Name)
					fa.FileClose()
				}
				// CHMOD event
				if 16 == event.Op {
					fmt.Printf("fa %s, watch %s receive event CHMOD\n", fa.Filename, event.Name)
				}
			}
		case err := <-watcher.Errors:
			log.Printf("%s receive error %s", fa.Filename, err.Error())
		case <-ticker.C:
			fa.Timeup()
		default:
			time.Sleep(time.Millisecond * 100)
		}
	}

	wg.Done()

	fmt.Printf("agent for %s is exiting...\n", fa.Filename)
}

/*
* FileOpen - open file while file agent initializing
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   nil, if succeed
*   error, if fail
 */
func (fa *FileAgent) FileOpen() error {
	// close old file
	if fa.File != nil {
		if err := fa.File.Close(); err != nil {
			log.Printf("file closing FAIL: %s", err.Error())
		}
	}

	fa.File = nil
	fa.FileInfo = nil

	// open new file
	filename := fa.Filename

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("file %s open FAIL: %s", fa.Filename, err.Error())
		return err
	}

	fileinfo, err := file.Stat()
	if err != nil {
		log.Printf("file %s stat FAIL: %s", fa.Filename, err.Error())
		return err
	}

	fmt.Printf("file %s is open\n", fa.Filename)

	fa.File = file
	fa.FileInfo = fileinfo
	fa.LastOffset = 0

	// seek the cursor to the end of new file
	_, err = fa.File.Seek(fa.FileInfo.Size(), os.SEEK_SET)
	if err != nil {
		log.Printf("seek file %s FAIL: %s", fa.Filename, err.Error())
	}
	fa.LastOffset += fa.FileInfo.Size()

	// set timestamp
	now := time.Now()
	minute := now.Format("200601021504")

	tsNow := now.Unix()
	tsStart := tsNow

	start, err := time.ParseInLocation("20060102150405", minute+"00", now.Location())
	if err != nil {
		log.Printf("timestamp setting FAIL: %s", err.Error())
	} else {
		tsStart = start.Unix()
	}

	for _, task := range fa.Tasks {
		task.TsStart = tsStart
		task.TsEnd = tsStart + task.Step - 1
		task.TsUpdate = tsNow
		task.ValueCnt = 0
		task.ValueMax = 0
		task.ValueMin = 1 << 32
		task.ValueSum = 0
	}

	return nil
}

/*
* FileInfoUpdate - stat file when WRITE
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   nil, if succeed
*   error, if fail
 */
func (fa *FileAgent) FileInfoUpdate() error {
	fileinfo, err := fa.File.Stat()
	if err != nil {
		log.Printf("file %s stat FAIL: %s", fa.Filename, err.Error())
		fa.UnchangeTime += 1
		return err
	}

	fa.FileInfo = fileinfo
	fa.UnchangeTime = 0
	return nil
}

/*
* FileClose - close file when REMOVE/RENAME
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   nil, if succeed
*   error, if fail
 */
func (fa *FileAgent) FileClose() error {
	// close old file
	if fa.File != nil {
		if err := fa.File.Close(); err != nil {
			log.Printf("file closing FAIL: %s", err.Error())
			return err
		}
	}

	fa.File = nil
	fa.FileInfo = nil
	fa.LastOffset = 0
	fa.UnchangeTime = 0

	for _, task := range fa.Tasks {
		task.TsStart = 0
		task.TsEnd = 0
		task.TsUpdate = 0
		task.ValueCnt = 0
		task.ValueMax = 0
		task.ValueMin = 1 << 32
		task.ValueSum = 0
	}

	return nil
}

/*
* FileReopen - reopen file when CREATE/ERROR
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   nil: succeed
*   error: fail
 */
func (fa *FileAgent) FileReopen() error {
	// close old file
	if fa.File != nil {
		if err := fa.File.Close(); err != nil {
			log.Printf("file closing FAIL: %s", err.Error())
		}
	}

	fa.File = nil
	fa.FileInfo = nil

	// open new file
	filename := fa.Filename

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("file %s opening FAIL: %s", fa.Filename, err.Error())
		return err
	}

	fileinfo, err := file.Stat()
	if err != nil {
		log.Printf("file %s stat FAIL: %s", fa.Filename, err.Error())
		return err
	}

	log.Printf("file %s recheck ok, it is a new file", fa.Filename)

	fa.File = file
	fa.FileInfo = fileinfo
	fa.LastOffset = 0

	// seek the cursor to the start of new file
	_, err = fa.File.Seek(0, os.SEEK_SET)
	if err != nil {
		log.Printf("seek file %s FAIL: %s", fa.Filename, err.Error())
	}

	// set timestamp
	now := time.Now()
	minute := now.Format("200601021504")

	tsNow := now.Unix()
	tsStart := tsNow

	start, err := time.ParseInLocation("20060102150405", minute+"00", now.Location())
	if err != nil {
		log.Printf("timestamp setting FAIL: %s", err.Error())
	} else {
		tsStart = start.Unix()
	}

	for _, task := range fa.Tasks {
		task.TsStart = tsStart
		task.TsEnd = tsStart + task.Step - 1
		task.TsUpdate = tsNow
		task.ValueCnt = 0
		task.ValueMax = 0
		task.ValueMin = 1 << 32
		task.ValueSum = 0
	}

	return nil
}

/*
* FileRecheck - recheck the file for file agent
*
* RECEIVER: *FileAgent
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   nil, if succeed
*   error, if fail
 */
func (fa *FileAgent) FileRecheck() error {
	filename := fa.Filename

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("file %s opening FAIL: %v", fa.Filename, err)
		fa.UnchangeTime = 0
		return err
	}

	fileinfo, err := file.Stat()
	if err != nil {
		log.Printf("file %s stat FAIL: %v", fa.Filename, err)
		fa.UnchangeTime = 0
		return err
	}

	isSameFile := os.SameFile(fa.FileInfo, fileinfo)
	if !isSameFile {
		log.Printf("file %s recheck, it is a new file", fa.Filename)
		if fa.File != nil {
			if err := fa.File.Close(); err != nil {
				log.Printf("old file closing FAIL: %v", err)
			}
		}

		fa.File = file
		fa.FileInfo = fileinfo
		fa.LastOffset = 0
		fa.UnchangeTime = 0

		// seek the cursor to the end of new file
		offset, err := fa.File.Seek(fa.FileInfo.Size(), os.SEEK_SET)
		if err != nil {
			log.Printf("seek file %s FAIL: %v", fa.Filename, err)
		}
		log.Printf("seek file %s to %d", fa.Filename, offset)
		fa.LastOffset += fa.FileInfo.Size()

		now := time.Now()
		minute := now.Format("200601021504")

		tsNow := now.Unix()
		tsStart := tsNow

		start, err := time.ParseInLocation("20060102150405", minute+"00", now.Location())
		if err != nil {
			log.Printf("timestamp setting FAIL: %v", err)
		} else {
			tsStart = start.Unix()
		}

		for _, task := range fa.Tasks {
			task.TsStart = tsStart
			task.TsEnd = tsStart + task.Step - 1
			task.TsUpdate = tsNow
			task.ValueCnt = 0
			task.ValueMax = 0
			task.ValueMin = 1 << 32
			task.ValueSum = 0
		}

		return nil
	} else {
		fa.UnchangeTime = 0
		return nil
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
		log.Printf("file %s stat FAIL: %v", err)
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

	// replace the FileInfo for reading the new content
	fa.UnchangeTime = 0
	fa.FileInfo = fileinfo
	return true
}
