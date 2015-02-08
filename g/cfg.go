package g

import (
	"encoding/json"
	"github.com/toolkits/file"
	"github.com/toolkits/net"
	"log"
	"sync"
)

const (
	VERSION = "1.0.1"
)

type RedisConfig struct {
	Dsn         string `json:"dsn"`
	MaxIdle     int    `json:"maxIdle"`
	RsPrefix    string `json:"rsPrefix"`
	CNamePrefix string `json:"cnamePrefix"`
}

type DBConfig struct {
	Dsn     string `json:"dsn"`
	MaxIdle int    `json:"maxIdle"`
}

type ScribeConfig struct {
	Ip   string `json:"ip"`
	Port int    `json:"port"`
}

type HttpConfig struct {
	Addr string `json:"addr"`
	Port int    `json:"port"`
}

type RpcConfig struct {
	Addr string `json:"addr"`
	Port int    `json:"port"`
}

type GlobalConfig struct {
	Debug      bool          `json:"debug"`
	Interval   int           `json:"interval"`
	DockerPort int           `json:"dockerPort"`
	Domain     string        `json:"domain"`
	LocalIp    string        `json:"localIp"`
	Redis      *RedisConfig  `json:"redis"`
	DB         *DBConfig     `json:"db"`
	Scribe     *ScribeConfig `json:"scribe"`
	Http       *HttpConfig   `json:"http"`
	Rpc        *RpcConfig    `json:"rpc"`
}

var (
	ConfigFile string
	config     *GlobalConfig
	configLock = new(sync.RWMutex)
)

func Config() *GlobalConfig {
	configLock.RLock()
	defer configLock.RUnlock()
	return config
}

func ParseConfig(cfg string) {
	if cfg == "" {
		log.Fatalln("use -c to specify configuration file")
	}

	if !file.IsExist(cfg) {
		log.Fatalln("config file:", cfg, "is not existent")
	}

	ConfigFile = cfg

	configContent, err := file.ToTrimString(cfg)
	if err != nil {
		log.Fatalln("read config file:", cfg, "fail:", err)
	}

	var c GlobalConfig
	err = json.Unmarshal([]byte(configContent), &c)
	if err != nil {
		log.Fatalln("parse config file:", cfg, "fail:", err)
	}

	if c.LocalIp == "" {
		// detect local ip
		localIps, err := net.IntranetIP()
		if err != nil {
			log.Fatalln("get intranet ip fail:", err)
		}

		if len(localIps) == 0 {
			log.Fatalln("no intranet ip found")
		}

		c.LocalIp = localIps[0]
	}

	if c.Http.Addr == "" {
		c.Http.Addr = c.LocalIp
	}

	if c.Rpc.Addr == "" {
		c.Rpc.Addr = c.LocalIp
	}

	configLock.Lock()
	defer configLock.Unlock()

	config = &c

	if config.Debug {
		log.Println("read config file:", cfg, "successfully")
	}
}
