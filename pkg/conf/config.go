package conf

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
	"path"
)

// ProxySource 代理源
// ProxySource.Type 类型
// ProxySource.FetchURL 加载链接
// ProxySource.TTL 过期时间 秒
type ProxySource struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	FetchURL string `json:"fetchURL"`
	TTL      int    `json:"ttl"`
}

// Rule 规则
type Rule struct {
	Host  string   `json:"host"`
	Proxy []string `json:"proxy"`
}

// Config 配置
// Config.PoolSize 池大小
type Config struct {
	ProxyHost    []string       `json:"proxyHost"`
	ProxySources []*ProxySource `json:"proxySources"`
}

func ReadFromFile(path string) (*Config, error) {
	// 先判断一下是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 不在就创建一个空JSON进去
		file, _ := os.Create(path)
		file.Write([]byte("{}"))
		file.Close()
	}
	// 打开文件
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("无法打开文件: %v", err)
	}
	defer file.Close()
	// 读取文件内容
	byteValue, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("读取文件时出错: %v", err)
	}
	// 解码 JSON 数据
	var c Config
	if err := json.Unmarshal(byteValue, &c); err != nil {
		log.Fatalf("解码 JSON 时出错: %v", err)
	}
	return &c, nil
}

func checkPath(p string) string {
	if path.IsAbs(p) {
		return p
	}
	dir, _ := os.Getwd()
	return path.Join(dir, p)
}

var Host string
var Port string
var IsDebug bool
var LogEnabled bool
var LogDirPath string
var ConfigPath string
var VersionOut bool

func init() {
	flag.StringVar(&Host, "host", "0.0.0.0", "host")
	flag.StringVar(&Port, "port", "8001", "port")
	flag.BoolVar(&IsDebug, "debug", false, "logout debug")
	flag.BoolVar(&LogEnabled, "log", false, "log")
	flag.BoolVar(&VersionOut, "version", false, "output version")
	flag.StringVar(&LogDirPath, "logDir", "log", "log path")
	flag.StringVar(&ConfigPath, "config", "conf.json", "config path")
	flag.Parse()
	LogDirPath = checkPath(LogDirPath)
	ConfigPath = checkPath(ConfigPath)
	if os.Getenv("PROXY_SERVER_DEBUG") == "true" {
		IsDebug = true
	}
}
