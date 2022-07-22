// Package psql Содержит реализацию интерфейса репозитория логов для postgresql
package psql

import (
	"github.com/n-r-w/lg"
	"github.com/n-r-w/postgres"
	"github.com/n-r-w/updsrv/internal/config"
)

type Repo struct {
	*postgres.Service
	config *config.Config
	cache  *Cache
	logger lg.Logger
}

func NewRepo(pg *postgres.Service, config *config.Config, logger lg.Logger) *Repo {
	r := &Repo{
		Service: pg,
		config:  config,
		logger:  logger,
	}
	r.cache = NewCache(r) // циклическая ссылка в go не приводит к утечке памяти
	return r
}
