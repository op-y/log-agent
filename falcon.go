/*
* falcon.go - the data structure of open falcon and related functions
*
* history
* --------------------
* 2017/8/18, by Ye Zhiqin, create
*
* DESCRIPTION
* This file contains the definition of open falcon data structure
* and the function to push data to open falcon
 */

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type FalconData struct {
	Metric      string      `json:"metric"`
	Endpoint    string      `json:"endpoint"`
	Value       interface{} `json:"value"`
	CounterType string      `json:"counterType"`
	Tags        string      `json:"tags"`
	Timestamp   int64       `json:"timestamp"`
	Step        int64       `json:"step"`
}

/*
* SetValue - set FalconData value
*
* RECEIVER: *FalconData
*
* PARAMS:
*   - v: value
*
* RETURNS:
*   No return value
 */
func (data *FalconData) SetValue(v interface{}) {
	data.Value = v
}

/*
* String - generate a new FalconData
*
* RECEIVER: *FalconData
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   - string: string to display
 */
func (data *FalconData) String() string {
	s := fmt.Sprintf("FalconData Metric:%s Endpoint:%s Value:%v CounterType:%s Tags:%s Timestamp:%d Step:%d",
		data.Metric, data.Endpoint, data.Value, data.CounterType, data.Tags, data.Timestamp, data.Step)
	return s
}

/*
* NewFalconData - generate a new FalconData
*
* PARAMS:
*   - metric
*   - endpoint
*   - value
*   - counterType
*   - timestamp
*   - step
*
* RETURNS:
*   - *FalconData
 */
func NewFalconData(metric string, endpoint string, value interface{}, counterType string, tags string, timestamp int64, step int64) *FalconData {
	point := &FalconData{
		Metric:      metric,
		Endpoint:    GetEndpoint(endpoint),
		CounterType: counterType,
		Tags:        tags,
		Timestamp:   GetTimestamp(timestamp),
		Step:        step,
	}
	point.SetValue(value)
	return point
}

/*
* GetEndpoint - generate endpoint value
*
* PARAMS:
*   - endpoint
*
* RETURNS:
*   - endpoint: if endpoint is avaliable
*   - hostname: if endpoint is empty
*   - localhost: if endpoint is empty and can't get hostname
 */
func GetEndpoint(endpoint string) string {
	if endpoint != "" {
		return endpoint
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	return hostname
}

/*
* GetTimestamp - generate timestamp value
*
* PARAMS:
*   - timestamp
*
* RETURNS:
*   - timestamp: if timestamp > 0
*   - now: if timestamp <= 0
 */
func GetTimestamp(timestamp int64) int64 {
	if timestamp > 0 {
		return timestamp
	} else {
		return time.Now().Unix()
	}
}

/*
* PushData - push data to open falcon
*
* PARAMS:
*   - api: url of agent or transfer
*   - data: an array of FalconData
*
* RETURNS:
*   - []byte, nil: if succeed
*   - nil, error: if fail
 */
func PushData(api string, data []*FalconData) ([]byte, error) {
	points, err := json.Marshal(data)
	if err != nil {
		log.Printf("data marshaling FAIL: %v", err)
		return nil, err
	}

	response, err := http.Post(api, "Content-Type: application/json", bytes.NewBuffer(points))
	if err != nil {
		log.Printf("api call FAIL: %v", err)
		return nil, err
	}
	defer response.Body.Close()

	content, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}
