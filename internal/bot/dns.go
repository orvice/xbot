package bot

import (
	"context"
	"fmt"
	"strings"

	"butterfly.orx.me/core/log"
	doh "github.com/babolivier/go-doh-client"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func dnsQueryHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)

	// Extract domain from message
	domain := strings.TrimPrefix(update.Message.Text, "/dns_query ")
	domain = strings.TrimSpace(domain)

	if domain == "" {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please provide a domain name. Usage: /dns_query example.com",
		})
		if err != nil {
			logger.Error("SendMessage error", "error", err)
		}
		return
	}

	resolver := doh.Resolver{
		Host:  "1.1.1.1",
		Class: doh.IN,
	}

	// Perform A record lookup
	records, _, err := resolver.LookupA(domain)
	if err != nil {
		logger.Error("LookupA error", "error", err, "domain", domain)
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Error looking up domain %s: %v", domain, err),
		})
		if err != nil {
			logger.Error("SendMessage error", "error", err)
		}
		return
	}

	// Format response
	var response strings.Builder
	response.WriteString(fmt.Sprintf("DNS query results for *%s*:\n\n", bot.EscapeMarkdown(domain)))
	for _, record := range records {
		response.WriteString(fmt.Sprintf("🌐 `%s`\n", record.IP.String()))
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      response.String(),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		logger.Error("SendMessage error", "error", err)
	}
}
