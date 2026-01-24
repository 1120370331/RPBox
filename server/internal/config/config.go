package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Storage  StorageConfig  `mapstructure:"storage"`
	OSS      OSSConfig      `mapstructure:"oss"`
	Backup   BackupConfig   `mapstructure:"backup"`
	Updater  UpdaterConfig  `mapstructure:"updater"`
	Redis    RedisConfig    `mapstructure:"redis"`
	SMTP     SMTPConfig     `mapstructure:"smtp"`
	CORS     CORSConfig     `mapstructure:"cors"`
	RateLimit RateLimitConfig `mapstructure:"rate_limit"`
}

type UpdaterConfig struct {
	LatestVersion string `mapstructure:"latest_version"`
	BaseURL       string `mapstructure:"base_url"`
	ReleaseNotes  string `mapstructure:"release_notes"`
	PubDate       string `mapstructure:"pub_date"`
	SignatureDir  string `mapstructure:"signature_dir"`
}

var globalConfig *Config

// Get 获取全局配置
func Get() *Config {
	return globalConfig
}

type StorageConfig struct {
	Path string `mapstructure:"path"`
}

type OSSConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Endpoint        string `mapstructure:"endpoint"`
	Bucket          string `mapstructure:"bucket"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Prefix          string `mapstructure:"prefix"`
}

type BackupConfig struct {
	Enabled         bool            `mapstructure:"enabled"`
	IntervalMinutes int             `mapstructure:"interval_minutes"`
	RetentionDays   int             `mapstructure:"retention_days"`
	LocalDir        string          `mapstructure:"local_dir"`
	Environment     string          `mapstructure:"environment"`
	PGDumpPath      string          `mapstructure:"pg_dump_path"`
	RunOnStart      bool            `mapstructure:"run_on_start"`
	TimeoutMinutes  int             `mapstructure:"timeout_minutes"`
	OSS             BackupOSSConfig `mapstructure:"oss"`
}

type BackupOSSConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	Endpoint         string `mapstructure:"endpoint"`
	InternalEndpoint string `mapstructure:"internal_endpoint"`
	UseInternal      bool   `mapstructure:"use_internal"`
	UseHTTPS         bool   `mapstructure:"use_https"`
	UseCname         bool   `mapstructure:"use_cname"`
	Bucket           string `mapstructure:"bucket"`
	AccessKeyID      string `mapstructure:"access_key_id"`
	AccessKeySecret  string `mapstructure:"access_key_secret"`
	Prefix           string `mapstructure:"prefix"`
}

type ServerConfig struct {
	Port          string `mapstructure:"port"`
	Mode          string `mapstructure:"mode"`
	MaxBodySizeMB int    `mapstructure:"max_body_size_mb"`
}

type DatabaseConfig struct {
	Host        string `mapstructure:"host"`
	Port        string `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	DBName      string `mapstructure:"dbname"`
	SSLMode     string `mapstructure:"sslmode"`
	SSLRootCert string `mapstructure:"sslrootcert"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	DevOrigins     []string `mapstructure:"dev_origins"`
}

type RateLimitConfig struct {
	Global RateLimitSetting `mapstructure:"global"`
	Auth   RateLimitSetting `mapstructure:"auth"`
	API    RateLimitSetting `mapstructure:"api"`
}

type RateLimitSetting struct {
	RPS   float64 `mapstructure:"rps"`
	Burst int     `mapstructure:"burst"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 默认值 - 使用相对于当前工作目录的路径
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.max_body_size_mb", 200)
	viper.SetDefault("storage.path", "storage") // 改为相对路径，不带 ./
	viper.SetDefault("database.sslmode", "require")
	viper.SetDefault("database.sslrootcert", "")
	viper.SetDefault("oss.enabled", false)
	viper.SetDefault("oss.prefix", "images")
	viper.SetDefault("backup.enabled", false)
	viper.SetDefault("backup.interval_minutes", 60)
	viper.SetDefault("backup.retention_days", 30)
	viper.SetDefault("backup.local_dir", "storage/backups")
	viper.SetDefault("backup.environment", "")
	viper.SetDefault("backup.pg_dump_path", "pg_dump")
	viper.SetDefault("backup.run_on_start", true)
	viper.SetDefault("backup.timeout_minutes", 60)
	viper.SetDefault("backup.oss.enabled", false)
	viper.SetDefault("backup.oss.use_internal", false)
	viper.SetDefault("backup.oss.use_https", true)
	viper.SetDefault("backup.oss.use_cname", false)
	viper.SetDefault("backup.oss.prefix", "db-backups")
	viper.SetDefault("cors.allowed_origins", []string{})
	viper.SetDefault("cors.dev_origins", []string{})
	viper.SetDefault("rate_limit.global.rps", 100)
	viper.SetDefault("rate_limit.global.burst", 200)
	viper.SetDefault("rate_limit.auth.rps", 1)
	viper.SetDefault("rate_limit.auth.burst", 3)
	viper.SetDefault("rate_limit.api.rps", 30)
	viper.SetDefault("rate_limit.api.burst", 60)

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	if err := mergeLocalConfig("config.local.yaml"); err != nil {
		return nil, err
	}
	if err := mergeLocalConfig(filepath.Join("config", "config.local.yaml")); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	globalConfig = &cfg
	return &cfg, nil
}

func mergeLocalConfig(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	viper.SetConfigFile(path)
	if err := viper.MergeInConfig(); err != nil {
		return err
	}
	return nil
}
