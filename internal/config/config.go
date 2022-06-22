// Package config ...
package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/n-r-w/lg"
)

// Config logserver.toml
type Config struct {
	SuperAdminID         int
	Host                 string   `toml:"HOST"`
	Port                 string   `toml:"PORT"`
	DatabaseURL          string   `toml:"DATABASE_URL"`
	MaxDbSessions        int      `toml:"MAX_DB_SESSIONS"`
	MaxDbSessionIdleTime int      `toml:"MAX_DB_SESSION_IDLE_TIME"`
	DbReadTimeout        int      `toml:"DB_READ_TIMEOUT"`
	DbWriteTimeout       int      `toml:"DB_WRITE_TIMEOUT"`
	HttpReadTimeout      int      `toml:"HTTP_READ_TIMEOUT"`
	HttpWriteTimeout     int      `toml:"HTTP_WRITE_TIMEOUT"`
	HttpShutdownTimeout  int      `toml:"HTTP_SHUTDOWN_TIMEOUT"`
	RateLimit            int      `toml:"RATE_LIMIT"`
	RateLimitBurst       int      `toml:"RATE_LIMIT_BURST"`
	MaxUpdateSize        int      `toml:"MAX_UPDATE_SIZE"`
	MaxVersionCount      int      `toml:"MAX_VERSION_COUNT"`
	Tokens               []string `toml:"TOKENS"`
}

const (
	superAdminID         = 1
	maxDbSessions        = 50
	maxDbSessionIdleTime = 50
	maxLogRecordsResult  = 100000
	defaultSessionAge    = 60 * 60 * 24 // 24 часа
)

// New Инициализация конфига значениями по умолчанию
func New(configPath string, logger lg.Logger) (*Config, error) {
	c := &Config{
		SuperAdminID:         superAdminID,
		Host:                 "0.0.0.0",
		Port:                 "8080",
		DatabaseURL:          "",
		MaxDbSessions:        maxDbSessions,
		MaxDbSessionIdleTime: maxDbSessionIdleTime,
		DbReadTimeout:        10,
		DbWriteTimeout:       5,
		HttpReadTimeout:      15,
		HttpWriteTimeout:     10,
		HttpShutdownTimeout:  10,
		RateLimit:            10000,
		RateLimitBurst:       20000,
		MaxUpdateSize:        200,
		MaxVersionCount:      100,
		Tokens:               []string{},
	}

	if configPath != "" {
		if _, err := toml.DecodeFile(configPath, c); err != nil {
			return nil, err
		}
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL undefined")
	}

	logger.Info("MAX_DB_SESSIONS: %d", c.MaxDbSessions)
	logger.Info("MAX_DB_SESSION_IDLE_TIME: %d", c.MaxDbSessionIdleTime)
	logger.Info("RATE_LIMIT: %d", c.RateLimit)
	logger.Info("RATE_LIMIT_BURST: %d", c.RateLimitBurst)
	logger.Info("DATABASE_URL: %s", c.DatabaseURL)

	return c, nil
}
