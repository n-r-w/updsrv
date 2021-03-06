// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package di

import (
	"github.com/n-r-w/httprouter"
	"github.com/n-r-w/lg"
	"github.com/n-r-w/postgres"
	"github.com/n-r-w/updsrv/internal/config"
	"github.com/n-r-w/updsrv/internal/presenter"
	"github.com/n-r-w/updsrv/internal/repo/psql"
)

// Injectors from wire.go:

// NewContainer - создание DI контейнера с помощью google wire
func NewContainer(logger lg.Logger, config2 *config.Config, dbUrl postgres.Url, dbOptions []postgres.Option) (*Container, func(), error) {
	service, err := postgres.New(dbUrl, logger, dbOptions...)
	if err != nil {
		return nil, nil, err
	}
	repo := psql.NewRepo(service, config2, logger)
	httprouterService := httprouter.New(logger)
	presenterService, err := presenter.New(httprouterService, repo, config2)
	if err != nil {
		return nil, nil, err
	}
	container := &Container{
		Logger:    logger,
		Config:    config2,
		DB:        service,
		Repo:      repo,
		Router:    httprouterService,
		Presenter: presenterService,
	}
	return container, func() {
	}, nil
}

// wire.go:

type Container struct {
	Logger    lg.Logger
	Config    *config.Config
	DB        *postgres.Service
	Repo      *psql.Repo
	Router    *httprouter.Service
	Presenter *presenter.Service
}
