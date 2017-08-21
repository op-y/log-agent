package main

import (
    "log"
    "regexp"
    "strconv"
    "time"
)

func MatchTs(line []byte, pattern string) (bool, time.Time, error) {
    re, err := regexp.Compile(pattern)
    if err != nil {
        log.Printf("re compiling ERROR: %v", err)
        return false, time.Now(), err
    }

    matches := re.FindSubmatch(line)

    if matches == nil {
        return false, time.Now(), nil
    }

    year   := string(matches[1])
    month  := string(matches[2])
    day    := string(matches[3])
    hour   := string(matches[4])
    minute := string(matches[5])
    second := string(matches[6])
    
    tsString := year+month+day+hour+minute+second
	ts, err := time.ParseInLocation("20060102150405", tsString, time.Now().Location())
	if err != nil {
		log.Printf("timestamp setting ERROR: %v", err)
	}

    return  true, ts, nil
}

func MatchKeyword(line []byte, pattern string) (bool, error) {
    re, err := regexp.Compile(pattern)
    if err != nil {
        log.Printf("re compiling ERROR: %v", err)
        return false, err
    }

    isMatch := re.Match(line)
    return isMatch, nil
}

func MatchCost(line []byte, pattern string) (bool, float64, error) {
    re, err := regexp.Compile(pattern)
    if err != nil {
        log.Printf("re compiling ERROR: %v", err)
        return false, 0, err
    }

    matches := re.FindSubmatch(line)

    if matches == nil {
        return false, 0, nil
    }

    cost, err := strconv.ParseFloat(string(matches[1]), 64)
    if err != nil {
        log.Printf("cost data string converting ERROR: %v", err)
        return true, 0, err
    }

    return  true, cost, nil
}
