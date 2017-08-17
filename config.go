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
    Items     []ItemConfig `yaml:"itmes"`
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

func LoadConfig() {
    buf, err := ioutil.ReadFile("config.yaml")
    if err != nil {
        log.Printf("configuration file reading ERROR: %v", err)
        return
    }

    if err := yaml.Unmarshal(buf, &config); err != nil {
        log.Fatalf("yaml file unmarshal ERROR: %v", err)
    }
    log.Printf("Config: %v", config)

}

func VerifyConfig() bool {
    return true
}

