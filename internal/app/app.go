// Package app ...
package app

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/n-r-w/httpserver"
	"github.com/n-r-w/lg"
	"github.com/n-r-w/postgres"
	"github.com/n-r-w/updsrv/internal/config"
	"github.com/n-r-w/updsrv/internal/di"
)

const version = "1.0.3"

func Start(cfg *config.Config, logger lg.Logger) {
	logger.Info("updsrv %s", version)

	// инициализация DI контейнера
	con, _, err := di.NewContainer(logger, cfg, postgres.Url(cfg.DatabaseURL),
		[]postgres.Option{
			postgres.MaxConns(cfg.MaxDbSessions),
			postgres.MaxMaxConnIdleTime(time.Duration(cfg.MaxDbSessionIdleTime) * time.Second),
		},
	)
	if err != nil {
		logger.Err(err)
		return
	}

	// запускаем http сервер
	httpServer := httpserver.New(con.Router.Handler(), logger,
		httpserver.Address(con.Config.Host, con.Config.Port),
		httpserver.ReadTimeout(time.Second*time.Duration(con.Config.HttpReadTimeout)),
		httpserver.WriteTimeout(time.Second*time.Duration(con.Config.HttpWriteTimeout)),
		httpserver.ShutdownTimeout(time.Second*time.Duration(con.Config.HttpShutdownTimeout)),
	)

	// ждем сигнал от сервера или нажатия ctrl+c
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case <-interrupt:
		logger.Info("shutdown, timeout %d ...", cfg.HttpShutdownTimeout)
	case err := <-httpServer.Notify():
		logger.Error("http server notification: %v", err)
	}

	// ждем завершения
	err = httpServer.Shutdown()
	if err != nil {
		logger.Error("shutdown error: %v", err)
	} else {
		logger.Info("shutdown ok")
	}

}
