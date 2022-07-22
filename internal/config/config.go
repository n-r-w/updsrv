// Package config ...
package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/n-r-w/lg"
)

// Config updsrv.toml
type Config struct {
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
	MinVersionAge        int      `toml:"MIN_VERSION_AGE"`
	TokensRead           []string `toml:"TOKENS_READ"`
	TokensWrite          []string `toml:"TOKENS_WRITE"`
}

const (
	maxDbSessions        = 50
	maxDbSessionIdleTime = 50
)

// New Инициализация конфига значениями по умолчанию
func New(configPath string, logger lg.Logger) (*Config, error) {
	c := &Config{
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
		MaxVersionCount:      30,
		MinVersionAge:        20,
		TokensRead:           []string{},
		TokensWrite:          []string{},
	}

	if configPath != "" {
		if _, err := toml.DecodeFile(configPath, c); err != nil {
			return nil, err
		}
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL undefined")
	}

	return c, nil
}
