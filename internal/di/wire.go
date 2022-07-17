//go:build wireinject
// +build wireinject

// Package di. Автоматическое внедрение зависимостей
package di

import (
	"github.com/google/wire"
	"github.com/n-r-w/httprouter"
	"github.com/n-r-w/lg"
	"github.com/n-r-w/postgres"
	"github.com/n-r-w/updsrv/internal/config"
	"github.com/n-r-w/updsrv/internal/presenter"
	"github.com/n-r-w/updsrv/internal/repo/psql"
)

type Container struct {
	Logger    lg.Logger
	Config    *config.Config
	DB        *postgres.Service
	Repo      *psql.Repo
	Router    *httprouter.Service
	Presenter *presenter.Service
}

// NewContainer - создание DI контейнера с помощью google wire
func NewContainer(logger lg.Logger, config *config.Config, dbUrl postgres.Url, dbOptions []postgres.Option) (*Container, func(), error) {
	panic(wire.Build(
		postgres.New,

		wire.Bind(new(presenter.UpdateInterface), new(*psql.Repo)),
		psql.NewRepo,

		wire.Bind(new(httprouter.Router), new(*httprouter.Service)),
		httprouter.New,

		presenter.New,

		wire.Struct(new(Container), "*"),
	))
}
