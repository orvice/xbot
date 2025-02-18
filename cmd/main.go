package main

import (
	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	"go.orx.me/xbot/internal/conf"
	"go.orx.me/xbot/internal/http"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
)

func NewApp() *app.App {
	app := core.New(&app.Config{
		Config:   conf.Conf,
		Service:  "api",
		Router:   http.Router,
		InitFunc: []func() error{},
	})
	return app
}

func main() {
	app := NewApp()
	app.Run()
}
