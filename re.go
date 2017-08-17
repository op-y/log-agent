package main

import (
    "log"
    "regexp"
    "strconv"
)

func MatchTs(line []byte, pattern string) (bool, string, error) {
    re, err := regexp.Compile(pattern)
    if err != nil {
        log.Printf("re compiling ERROR: %v", err)
        return false, "", err
    }

    matches := re.FindSubmatch(line)

    if matches == nil {
        return false, "", nil
    }

    year   := string(matches[1])
    month  := string(matches[2])
    day    := string(matches[3])
    hour   := string(matches[4])
    minute := string(matches[5])
    
    ts := year+month+day+hour+minute

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
