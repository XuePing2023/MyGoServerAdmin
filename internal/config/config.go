// Package config 负责加载与暴露系统配置。
// 优先级：环境变量（SA_ 前缀） > config.yaml > 内置默认值。
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App     AppConfig     `mapstructure:"app"`
	Log     LogConfig     `mapstructure:"log"`
	Mongo   MongoConfig   `mapstructure:"mongo"`
	Redis   RedisConfig   `mapstructure:"redis"`
	Cache   CacheConfig   `mapstructure:"cache"`
	JWT     JWTConfig     `mapstructure:"jwt"`
	CORS    CORSConfig    `mapstructure:"cors"`
	Captcha CaptchaConfig `mapstructure:"captcha"`
	OpLog   OpLogConfig   `mapstructure:"oplog"`
	Seed    SeedConfig    `mapstructure:"seed"`
}

type AppConfig struct {
	Name         string `mapstructure:"name"`
	Version      string `mapstructure:"version"`
	Env          string `mapstructure:"env"`
	Port         int    `mapstructure:"port"`
	ReadTimeout  int    `mapstructure:"readTimeout"`
	WriteTimeout int    `mapstructure:"writeTimeout"`
	UploadDir    string `mapstructure:"uploadDir"`
	MaxUploadMB  int    `mapstructure:"maxUploadMB"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSizeMB  int    `mapstructure:"maxSizeMB"`
	MaxBackups int    `mapstructure:"maxBackups"`
	MaxAgeDays int    `mapstructure:"maxAgeDays"`
}

type MongoConfig struct {
	URI            string `mapstructure:"uri"`
	Database       string `mapstructure:"database"`
	TimeoutSeconds int    `mapstructure:"timeoutSeconds"`
}

// RedisConfig Redis 连接配置，enabled=false 时不启用缓存。
type RedisConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// CacheConfig 缓存策略 TTL（秒），按策略区分，避免全部一样。
type CacheConfig struct {
	PublicTTLSeconds int `mapstructure:"publicTTLSeconds"` // PUBLIC：全用户共享（系统参数/字典）
	UserTTLSeconds   int `mapstructure:"userTTLSeconds"`   // USER：按用户区分（个人资料/工作台）
	RoleTTLSeconds   int `mapstructure:"roleTTLSeconds"`   // ROLE：按角色区分（菜单/权限）
}

type JWTConfig struct {
	Secret              string `mapstructure:"secret"`
	Issuer              string `mapstructure:"issuer"`
	AccessExpireMinutes int    `mapstructure:"accessExpireMinutes"`
	RefreshExpireHours  int    `mapstructure:"refreshExpireHours"`
}

type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allowOrigins"`
}

type CaptchaConfig struct {
	Enabled       bool `mapstructure:"enabled"`
	ExpireSeconds int  `mapstructure:"expireSeconds"`
}

type OpLogConfig struct {
	Enabled      bool `mapstructure:"enabled"`
	BodyMaxBytes int  `mapstructure:"bodyMaxBytes"`
}

type SeedConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// Load 从 dir（或 SA_CONFIG 指定的文件）加载配置，任何一层缺失都回退到默认值，
// 保证“零配置”也能启动。
func Load() *Config {
	v := viper.New()
	setDefaults(v)

	v.SetEnvPrefix("SA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	path := os.Getenv("SA_CONFIG")
	if path == "" {
		v.AddConfigPath(".")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
	} else {
		v.SetConfigFile(path)
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && path != "" {
			// 显式指定的配置文件读取失败时直接报错；缺省文件则使用默认值
			if !os.IsNotExist(err) {
				panic("读取配置文件失败: " + err.Error())
			}
		}
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		panic("解析配置失败: " + err.Error())
	}
	cfg.normalize()
	return cfg
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "ServerAdmin")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.env", "dev")
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.readTimeout", 30)
	v.SetDefault("app.writeTimeout", 30)
	v.SetDefault("app.uploadDir", "./uploads")
	v.SetDefault("app.maxUploadMB", 20)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.file", "logs/server.log")
	v.SetDefault("log.maxSizeMB", 100)
	v.SetDefault("log.maxBackups", 7)
	v.SetDefault("log.maxAgeDays", 30)

	v.SetDefault("mongo.uri", "mongodb://localhost:27017")
	v.SetDefault("mongo.database", "server_admin")
	v.SetDefault("mongo.timeoutSeconds", 10)

	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.addr", "127.0.0.1:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	v.SetDefault("cache.publicTTLSeconds", 600)
	v.SetDefault("cache.userTTLSeconds", 120)
	v.SetDefault("cache.roleTTLSeconds", 300)

	v.SetDefault("jwt.secret", "server-admin-secret-please-change-me")
	v.SetDefault("jwt.issuer", "server-admin")
	v.SetDefault("jwt.accessExpireMinutes", 120)
	v.SetDefault("jwt.refreshExpireHours", 168)

	v.SetDefault("cors.allowOrigins", []string{"*"})

	v.SetDefault("captcha.enabled", true)
	v.SetDefault("captcha.expireSeconds", 300)

	v.SetDefault("oplog.enabled", true)
	v.SetDefault("oplog.bodyMaxBytes", 2048)

	v.SetDefault("seed.enabled", true)
}

func (c *Config) normalize() {
	if c.App.UploadDir != "" && !filepath.IsAbs(c.App.UploadDir) {
		c.App.UploadDir, _ = filepath.Abs(c.App.UploadDir)
	}
	if c.JWT.AccessExpireMinutes <= 0 {
		c.JWT.AccessExpireMinutes = 120
	}
	if c.JWT.RefreshExpireHours <= 0 {
		c.JWT.RefreshExpireHours = 168
	}
	if c.App.MaxUploadMB <= 0 {
		c.App.MaxUploadMB = 20
	}
	if c.Mongo.TimeoutSeconds <= 0 {
		c.Mongo.TimeoutSeconds = 10
	}
	if c.Redis.Addr == "" {
		c.Redis.Addr = "127.0.0.1:6379"
	}
	if c.Cache.PublicTTLSeconds <= 0 {
		c.Cache.PublicTTLSeconds = 600
	}
	if c.Cache.UserTTLSeconds <= 0 {
		c.Cache.UserTTLSeconds = 120
	}
	if c.Cache.RoleTTLSeconds <= 0 {
		c.Cache.RoleTTLSeconds = 300
	}
	if c.OpLog.BodyMaxBytes <= 0 {
		c.OpLog.BodyMaxBytes = 2048
	}
	if c.Captcha.ExpireSeconds <= 0 {
		c.Captcha.ExpireSeconds = 300
	}
	if c.App.Name == "" {
		c.App.Name = "ServerAdmin"
	}
}

// IsProd 是否生产环境。
func (c *Config) IsProd() bool { return strings.EqualFold(c.App.Env, "prod") }
