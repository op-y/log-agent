package main

import (
    //"bytes"
    "fmt"
    "io"
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
    TsStart     int64
    TsEnd       int64
	TsUpdate    int64
	Pattern     string
	Method      string
	ValueCnt    int64
	ValueMax    float64
	ValueMin    float64
	ValueAvg    float64
	ValueSum    float64
}

func (fa *FileAgent) TimeupAgent() {
    //log.Printf("file %s ticker timeup", fa.Filename)
}

func (fa *FileAgent) isChanged() bool {
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
        (! thisTailable || lastSize == thisSize) &&
        lastModTime == thisModTime {
        
        fa.UnchangeTime += 1
        return false
    }

    log.Printf("file %s is changed", fa.Filename)
    fa.UnchangeTime = 0
    fa.FileInfo = fileinfo
    return true
}

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
    if ! isNewFile {
        log.Printf("file %s recheck, it is a new file", fa.Filename)
        if fa.File != nil {
            if err:= fa.File.Close(); err != nil {
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

        start, err := time.Parse("20060102150405", minute+"00")
        if err != nil {
            log.Printf("timestamp setting ERROR: %v", err)
        } else {
            tsStart = start.Unix()
        }

        end, err := time.Parse("20060102150405", minute+"59")
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
            task.ValueMin = 0
            task.ValueAvg = 0
            task.ValueSum = 0
        }

        return nil
    } else {
        //log.Printf("recheck file, but it not changed")
        fa.UnchangeTime = 0
        return nil
    }
}

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

    // read data
    log.Printf("file %s, size: %d --- offset: %d", fa.Filename, fa.FileInfo.Size(), fa.LastOffset)

    bufsize := size - fa.LastOffset
    log.Printf("buffer size: %d", bufsize)
    if bufsize == 0 {
        log.Printf("file %s changed, but its size is not changed", fa.Filename)
        return
    }
    data := make([]byte, bufsize)

    readsize, err := fa.File.Read(data)
    log.Printf("real read size: %d", readsize)

    if err != nil && err != io.EOF {
        log.Printf("file %s read ERROR: %v", err)
        return
    }
    if readsize == 0 {
        log.Printf("file %s read 0 data", fa.Filename)
        return
    }

    log.Printf("=======DATA========")
    fmt.Printf("%s", string(data))
    log.Printf("===================")

    // Read 本身会移动File偏移量,这里不需要再Seek,而是应该行处理后考虑是否回调偏移量
    //offset, err := fa.File.Seek(int64(readsize), os.SEEK_CUR)
    //log.Printf("File(seek) %s offset %d", fa.Filename, offset)
    //if err != nil {
    //    log.Printf("file %s seek ERROR: %v", fa.Filename, err)
    //}
    fa.LastOffset += int64(readsize)

    log.Printf("file %s, size: %d --- offset: %d", fa.Filename, fa.FileInfo.Size(), fa.LastOffset)
    return

    //sep := []byte(fa.Delimiter)
    //lines := bytes.SplitAfter(data, sep)
    //length := len(lines) 
    //avaliableSize := 0
    //if length < 2 {
    //    return
    //} else {
    //    lines = lines[0:length-2]
    //    for _, line := range lines {
    //        avaliableSize += len(line)
    //        log.Printf("===================")
    //        fmt.Printf("%s", string(line))
    //        log.Printf("===================")
    //    }
    //    offset, err := fa.File.Seek(int64(avaliableSize), os.SEEK_CUR)
    //    if err != nil {
    //        log.Printf("file %s seek ERROR: %v", fa.Filename, err)
    //    }
    //    if offset != int64(avaliableSize) {
    //        log.Printf("offset is not equal avaliableSize")
    //    }
    //    fa.LastOffset += int64(avaliableSize)
    //    return
    //}
}

func TryReading(fa *FileAgent) {
    if fa.File == nil {
        log.Printf("file %s is nil", fa.Filename)
        if err := fa.Recheck(); err != nil {
            log.Printf("file recheck ERROR: %v", err)
        }
        return
    }

    if ! fa.isChanged() {
        //log.Printf("file %s is not changed", fa.Filename)
        if fa.UnchangeTime >= MAX_UNCHANGED_TIME {
            fa.Recheck()
        }
        return
    }

    //log.Printf("file %s is trying to read", fa.Filename)
    fa.ReadRemainder()
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
            TryReading(fa)
		}
	}

	wg.Done()
	log.Printf("wg: %v", wg)
	log.Printf("agent for %s exit...", fa.Filename)
}
