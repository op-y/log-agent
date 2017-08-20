package main

import (
	"crypto/md5"
	"io"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
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

func CheckConfigMD5() []byte {
	f, err := os.Open("config.yaml")
	if err != nil {
		log.Printf("configuration file opening ERROR: %v", err)
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Printf("md5 copying ERROR: %v", err)
	}
	return h.Sum(nil)
}

func LoadConfig() *Config {
	cfg := new(Config)

	buf, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Printf("configuration file reading ERROR: %v", err)
		return nil
	}

	if err := yaml.Unmarshal(buf, cfg); err != nil {
		log.Printf("yaml file unmarshal ERROR: %v", err)
		return nil
	}
	log.Printf("Config: %v", cfg)

	return cfg
}
