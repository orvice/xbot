package http

import (
	"github.com/gin-gonic/gin"
	"go.orx.me/xbot/internal/bot"
)

func Router(m *gin.Engine) {
	m.Any("/v1/webhook", func(c *gin.Context) {
		bot.GetBot().WebhookHandler().ServeHTTP(c.Writer, c.Request)
	})
}
