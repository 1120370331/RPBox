package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	Storage    StorageConfig    `mapstructure:"storage"`
	OSS        OSSConfig        `mapstructure:"oss"`
	Backup     BackupConfig     `mapstructure:"backup"`
	Updater    UpdaterConfig    `mapstructure:"updater"`
	CurseForge CurseForgeConfig `mapstructure:"curseforge"`
	TRP3Addons TRP3AddonsConfig `mapstructure:"trp3_addons"`
	Redis      RedisConfig      `mapstructure:"redis"`
	SMTP       SMTPConfig       `mapstructure:"smtp"`
	CORS       CORSConfig       `mapstructure:"cors"`
	RateLimit  RateLimitConfig  `mapstructure:"rate_limit"`
}

type UpdaterConfig struct {
	LatestVersion string               `mapstructure:"latest_version"`
	BaseURL       string               `mapstructure:"base_url"`
	ReleaseNotes  string               `mapstructure:"release_notes"`
	PubDate       string               `mapstructure:"pub_date"`
	SignatureDir  string               `mapstructure:"signature_dir"`
	Beta          DesktopUpdaterConfig `mapstructure:"beta"`
	Mobile        MobileUpdaterConfig  `mapstructure:"mobile"`
}

type DesktopUpdaterConfig struct {
	LatestVersion string `mapstructure:"latest_version"`
	BaseURL       string `mapstructure:"base_url"`
	ReleaseNotes  string `mapstructure:"release_notes"`
	PubDate       string `mapstructure:"pub_date"`
}

type MobileUpdaterConfig struct {
	Android MobilePlatformUpdaterConfig `mapstructure:"android"`
	IOS     MobilePlatformUpdaterConfig `mapstructure:"ios"`
}

type MobilePlatformUpdaterConfig struct {
	LatestVersion string `mapstructure:"latest_version"`
	URL           string `mapstructure:"url"`
	ReleaseNotes  string `mapstructure:"release_notes"`
	PubDate       string `mapstructure:"pub_date"`
	Mandatory     bool   `mapstructure:"mandatory"`
}

type CurseForgeConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	APIKey          string `mapstructure:"api_key"`
	BaseURL         string `mapstructure:"base_url"`
	TimeoutSeconds  int    `mapstructure:"timeout_seconds"`
	CacheTTLMinutes int    `mapstructure:"cache_ttl_minutes"`
}

type TRP3AddonsConfig struct {
	Enabled                     bool   `mapstructure:"enabled"`
	GitHubAPIBaseURL            string `mapstructure:"github_api_base_url"`
	GitHubDownloadMirrorBaseURL string `mapstructure:"github_download_mirror_base_url"`
	GitHubToken                 string `mapstructure:"github_token"`
	ProxyURL                    string `mapstructure:"proxy_url"`
	CacheEnabled                bool   `mapstructure:"cache_enabled"`
	CacheSubdir                 string `mapstructure:"cache_subdir"`
	MaxDownloadMB               int    `mapstructure:"max_download_mb"`
	TimeoutSeconds              int    `mapstructure:"timeout_seconds"`
	CacheTTLMinutes             int    `mapstructure:"cache_ttl_minutes"`
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
	Port           string   `mapstructure:"port"`
	Mode           string   `mapstructure:"mode"`
	MaxBodySizeMB  int      `mapstructure:"max_body_size_mb"`
	ApiHost        string   `mapstructure:"api_host"` // API 基础 URL，如 https://api.rpbox.app
	TrustedProxies []string `mapstructure:"trusted_proxies"`
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

	if err := loadDotEnv(".env"); err != nil {
		return nil, err
	}
	if err := loadDotEnv(filepath.Join("config", ".env")); err != nil {
		return nil, err
	}
	if err := loadDotEnv(filepath.Join("server", ".env")); err != nil {
		return nil, err
	}

	// 默认值 - 使用相对于当前工作目录的路径
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.max_body_size_mb", 200)
	viper.SetDefault("server.trusted_proxies", []string{})
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
	viper.SetDefault("curseforge.enabled", true)
	viper.SetDefault("curseforge.api_key", "")
	viper.SetDefault("curseforge.base_url", "https://api.curseforge.com")
	viper.SetDefault("curseforge.timeout_seconds", 10)
	viper.SetDefault("curseforge.cache_ttl_minutes", 360)
	viper.SetDefault("trp3_addons.enabled", true)
	viper.SetDefault("trp3_addons.github_api_base_url", "https://api.github.com")
	viper.SetDefault("trp3_addons.github_download_mirror_base_url", "")
	viper.SetDefault("trp3_addons.github_token", "")
	viper.SetDefault("trp3_addons.proxy_url", "")
	viper.SetDefault("trp3_addons.cache_enabled", true)
	viper.SetDefault("trp3_addons.cache_subdir", "cache/addons/trp3")
	viper.SetDefault("trp3_addons.max_download_mb", 256)
	viper.SetDefault("trp3_addons.timeout_seconds", 15)
	viper.SetDefault("trp3_addons.cache_ttl_minutes", 360)

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("curseforge.api_key", "CURSEFORGE_API_KEY")
	_ = viper.BindEnv("jwt.secret", "JWT_SECRET")
	_ = viper.BindEnv("trp3_addons.github_token", "TRP3_ADDONS_GITHUB_TOKEN", "GITHUB_TOKEN")
	_ = viper.BindEnv("trp3_addons.github_download_mirror_base_url", "TRP3_ADDONS_GITHUB_DOWNLOAD_MIRROR_BASE_URL")
	_ = viper.BindEnv("trp3_addons.proxy_url", "TRP3_ADDONS_PROXY_URL", "HTTPS_PROXY", "HTTP_PROXY")

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
	if err := validateJWTConfig(cfg.Server.Mode, cfg.JWT.Secret); err != nil {
		return nil, err
	}

	globalConfig = &cfg
	return &cfg, nil
}

func validateJWTConfig(serverMode, secret string) error {
	if !strings.EqualFold(strings.TrimSpace(serverMode), "release") {
		return nil
	}
	if len(strings.TrimSpace(secret)) < 32 {
		return fmt.Errorf("jwt.secret must contain at least 32 UTF-8 bytes when server.mode is release")
	}
	if isPlaceholderJWTSecret(secret) {
		return fmt.Errorf("jwt.secret must not use an example, placeholder, development, or test value when server.mode is release")
	}
	return nil
}

func isPlaceholderJWTSecret(secret string) bool {
	normalized := strings.ToLower(strings.TrimSpace(secret))
	knownPlaceholders := map[string]struct{}{
		"your-secret-key-change-in-production": {},
		"change-me-in-production":              {},
		"change-this-in-production":            {},
		"replace-me-with-a-secure-secret":      {},
		"replace-with-a-secure-secret":         {},
		"example-jwt-secret":                   {},
		"placeholder-jwt-secret":               {},
		"development-jwt-secret":               {},
		"test-jwt-secret":                      {},
	}
	if _, found := knownPlaceholders[normalized]; found {
		return true
	}
	weakMarkers := map[string]struct{}{
		"change":      {},
		"default":     {},
		"dev":         {},
		"development": {},
		"example":     {},
		"insecure":    {},
		"local":       {},
		"placeholder": {},
		"replace":     {},
		"sample":      {},
		"temp":        {},
		"temporary":   {},
		"test":        {},
		"testing":     {},
		"unsafe":      {},
		"your":        {},
	}
	placeholderVocabulary := map[string]struct{}{
		"a": {}, "change": {}, "default": {}, "dev": {}, "development": {},
		"do": {}, "example": {}, "for": {}, "in": {}, "insecure": {},
		"jwt": {}, "key": {}, "local": {}, "me": {}, "not": {},
		"only": {}, "placeholder": {}, "please": {}, "prod": {}, "production": {},
		"replace": {}, "sample": {}, "secret": {}, "secure": {}, "temp": {},
		"temporary": {}, "test": {}, "testing": {}, "this": {}, "token": {},
		"unsafe": {}, "use": {}, "value": {}, "with": {}, "your": {},
	}
	tokens := strings.FieldsFunc(normalized, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	containsWeakMarker := false
	usesOnlyPlaceholderVocabulary := len(tokens) > 0
	for _, token := range tokens {
		if _, found := weakMarkers[token]; found {
			containsWeakMarker = true
		}
		if _, found := placeholderVocabulary[token]; !found {
			usesOnlyPlaceholderVocabulary = false
		}
	}
	if containsWeakMarker && usesOnlyPlaceholderVocabulary {
		return true
	}

	// Padding an obvious weak value must not make it production-safe.
	compact := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\r', '\n', '-', '_', '.', ':', '/', '\\':
			return -1
		default:
			return r
		}
	}, normalized)
	for _, weak := range []string{"example", "placeholder", "development", "dev", "test", "testing", "changeme", "replaceme"} {
		if compact == strings.Repeat(weak, len(compact)/len(weak)) && len(compact)%len(weak) == 0 {
			return true
		}
	}
	return false
}

func loadDotEnv(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}
	return nil
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
