/*
* re.go - functions related to regular expression matching
*
* history
* --------------------
* 2017/8/18, by Ye Zhiqin, create
*
* DESCRIPTION
* This file contains three functions related to regular expression matching
* MatchTs - match and extract timestamp in log
* MatchKeyword - match the keyword in log
* MatchCost - match and extract cost value in log
 */

package main

import (
	"log"
	"regexp"
	"strconv"
	"time"
)

/*
* MatchTs - match and extract timestamp in log
*
* PARAMS:
*   - line: one line of log
*   - pattern: regular expression
*
* RETURNS:
*   - true, timestamp, nil: if match
*   - false, timestamp, nil: if not match
*   - false, timestamp, error: if fail
 */
func MatchTs(line []byte, pattern string) (bool, time.Time, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Printf("re compiling FAIL: %v", err)
		return false, time.Now(), err
	}

	matches := re.FindSubmatch(line)

	if matches == nil {
		return false, time.Now(), nil
	}

	year := string(matches[1])
	month := string(matches[2])
	day := string(matches[3])
	hour := string(matches[4])
	minute := string(matches[5])
	second := string(matches[6])

	tsString := year + month + day + hour + minute + second
	ts, err := time.ParseInLocation("20060102150405", tsString, time.Now().Location())
	if err != nil {
		log.Printf("timestamp setting FAIL: %v", err)
	}

	return true, ts, nil
}

/*
* MatchKeyword - match the keyword in log
*
* PARAMS:
*   - line: one line of log
*   - pattern: regular expression
*
* RETURNS:
*   - true, nil: if match
*   - false, nil: if not match
*   - false, error: if fail
 */
func MatchKeyword(line []byte, pattern string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Printf("re compiling FAIL: %v", err)
		return false, err
	}

	isMatch := re.Match(line)
	return isMatch, nil
}

/*
* MatchCost - match and extract cost value in log
*
* PARAMS:
*   - line: one line of log
*   - pattern: regular expression
*
* RETURNS:
*   - true, value, nil: if match
*   - false, 0, nil: if not match
*   - false, 0, error: if fail
 */
func MatchCost(line []byte, pattern string) (bool, float64, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Printf("re compiling FAIL: %v", err)
		return false, 0, err
	}

	matches := re.FindSubmatch(line)

	if matches == nil {
		return false, 0, nil
	}

	cost, err := strconv.ParseFloat(string(matches[1]), 64)
	if err != nil {
		log.Printf("cost data string converting FAIL: %v", err)
		return true, 0, err
	}

	return true, cost, nil
}
