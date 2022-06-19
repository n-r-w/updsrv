package main

import (
	"flag"

	"github.com/n-r-w/lg"
	"github.com/n-r-w/updsrv/internal/app"
	"github.com/n-r-w/updsrv/internal/config"
)

func main() {
	var configPath string
	// описание флагов командной строки
	flag.StringVar(&configPath, "config-path", "", "path to config file")

	// обработка командной строки
	flag.Parse()

	log := lg.New()

	// читаем конфиг
	cfg, err := config.New(configPath, log)
	if err != nil {
		log.Fatal("read config error: %v", err)

		return
	}

	app.Start(cfg, log)
}
