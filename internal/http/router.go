package http

import (
	"github.com/gin-gonic/gin"

	"butterfly.orx.me/core/log"
	"go.orx.me/xbot/internal/bot"
)

func Router(m *gin.Engine) {
	m.Any("/v1/webhook", func(c *gin.Context) {
		logger := log.FromContext(c.Request.Context())
		logger.Debug("new webhook request",
			"header", c.Request.Header,
			"method", c.Request.Method,
		)
		bot.GetBot().WebhookHandler().ServeHTTP(c.Writer, c.Request)
	})
}
