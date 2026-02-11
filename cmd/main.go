package main

import (
	"context"

	"butterfly.orx.me/core"
	"butterfly.orx.me/core/app"
	"go.orx.me/xbot/internal/bot"
	"go.orx.me/xbot/internal/conf"
	"go.orx.me/xbot/internal/dao"
	"go.orx.me/xbot/internal/http"
	"go.orx.me/xbot/internal/pkg/gemini"
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
			// Wrap dao.Init to match the required function signature
			func() error {
				return dao.Init(context.Background())
			},
			openai.Init,
			bot.Init,
			gemini.Init,
		},
	})
	return app
}

func main() {
	app := NewApp()
	app.Run()
}
