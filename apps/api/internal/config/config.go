package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Database    DatabaseConfig    `mapstructure:"database"`
	HTTP        HTTPConfig        `mapstructure:"http"`
	Logger      LoggerConfig      `mapstructure:"logger"`
	S3          S3Config          `mapstructure:"s3"`
	Invitations InvitationsConfig `mapstructure:"invitations"`
}

type InvitationsConfig struct {
	RequireInviteCode bool `mapstructure:"require_invite_code"`
	DefaultQuota      int  `mapstructure:"default_quota"` // initial invites per new user
}

type S3Config struct {
	Bucket          string        `mapstructure:"bucket"`
	Region          string        `mapstructure:"region"`
	Endpoint        string        `mapstructure:"endpoint"`
	AccessKeyID     string        `mapstructure:"access_key_id"`
	SecretAccessKey string        `mapstructure:"secret_access_key"`
	PresignExpiry   time.Duration `mapstructure:"presign_expiry"`
	CDNBaseURL      string        `mapstructure:"cdn_base_url"` // e.g. "https://cdn.skjul.com"
}

type DatabaseConfig struct {
	DatabaseURL     string        `mapstructure:"database_url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

type HTTPConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", "30m")
	v.SetDefault("database.conn_max_idle_time", "5m")

	v.SetDefault("http.host", "0.0.0.0")
	v.SetDefault("http.port", 8080)
	v.SetDefault("http.read_timeout", "10s")
	v.SetDefault("http.write_timeout", "10s")
	v.SetDefault("http.idle_timeout", "120s")

	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")

	v.SetDefault("s3.presign_expiry", "15m")

	v.SetDefault("invitations.require_invite_code", false)
	v.SetDefault("invitations.default_quota", 5)

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}
