/*
* config.go - the structure of configuration and related functions
*
* history
* --------------------
* 2017/8/18, by Ye Zhiqin, create
*
* DESCRIPTION
* This file contains the definition of configuration data structure
* and the functions to load configuration file and check md5sum of
* the configuration file
 */

package main

import (
	"crypto/md5"
	"io"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	CONFIG_CHECK_INTERVAL = 5
)

type Config struct {
	Falcon FalconConfig `yaml:"falcon"`
	Logs   []LogConfig  `yaml:"logs"`
}

type FalconConfig struct {
	Url      string `yaml:"url"`
	Endpoint string `yaml:"endpoint"`
}

type LogConfig struct {
	Name      string       `yaml:"name"`
	Path      string       `yaml:"path"`
	Delimiter string       `yaml:"delimiter"`
	TsEnabled bool         `yaml:"tsEnabled"`
	TsPattern string       `yaml:"tsPattern"`
	Items     []ItemConfig `yaml:"items"`
}

type ItemConfig struct {
	Metric      string `yaml:"metric"`
	Tags        string `yaml:"tags"`
	CounterType string `yaml:"counterType"`
	Step        int64  `yaml:"step"`
	Pattern     string `yaml:"pattern"`
	Method      string `yaml:"method"`
}

var config *Config
var configMD5Sum []byte

/*
* CheckConfigMD5 - calculate the md5sum of configuration file
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   []byte, the md5sum of configuration file
 */
func CheckConfigMD5() []byte {
	f, err := os.Open("config.yaml")
	if err != nil {
		log.Printf("configuration file opening FAIL: %v", err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Printf("md5 copying FAIL: %v", err)
	}
	return h.Sum(nil)
}

/*
* LoadConfig - load configuration file to Config struct
*
* PARAMS:
*   No paramter
*
* RETURNS:
*   nil, if error ocurred
*   *Config, if succeed
 */
func LoadConfig() *Config {
	cfg := new(Config)

	buf, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Printf("configuration file reading FAIL: %v", err)
		return nil
	}

	if err := yaml.Unmarshal(buf, cfg); err != nil {
		log.Printf("yaml file unmarshal FAIL: %v", err)
		return nil
	}
	log.Printf("config: %v", cfg)

	// check configuration
	if cfg.Falcon.Url == "" {
		log.Printf("Url of falcon agent api should not EMPTY!")
		return nil
	}

	for _, one := range cfg.Logs {
		if one.Name == "" {
			log.Printf("Name of log should not EMPTY!")
			return nil
		}
		if one.Path == "" {
			log.Printf("Path of log should not EMPTY!")
			return nil
		}

		for _, item := range one.Items {
			if item.Metric == "" {
				log.Printf("Metric of item should not EMPTY!")
				return nil
			}

			if item.CounterType != "GAUGE" && item.CounterType != "COUNTER" {
				log.Printf("CouterType of item should be 'GAUGE' or 'COUNTER'")
				return nil
			}

			if item.Pattern == "" {
				log.Printf("Pattern of item should not EMPTY!")
				return nil
			}

			if item.Method != "count" && item.Method != "statistic" {
				log.Printf("Method of item should be 'count' or 'statistic'")
				return nil
			}
		}
	}

	return cfg
}
