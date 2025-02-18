package main

import (
	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	"go.orx.me/xbot/internal/bot"
	"go.orx.me/xbot/internal/conf"
	"go.orx.me/xbot/internal/dao"
	"go.orx.me/xbot/internal/http"
	"go.orx.me/xbot/internal/pkg/openai"

	// mysql driver
	_ "github.com/go-sql-driver/mysql"
)

func NewApp() *app.App {
	app := core.New(&app.Config{
		Config:  conf.Conf,
		Service: "xbot",
		Router:  http.Router,
		InitFunc: []func() error{
			dao.Init,
			openai.Init,
			bot.Init,
		},
	})
	return app
}

func main() {
	app := NewApp()
	app.Run()
}
