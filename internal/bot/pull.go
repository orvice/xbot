package bot

import (
	"context"
	"time"

	"butterfly.orx.me/core/log"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.orx.me/xbot/internal/dao"
)

type PullConfig struct {
	Type    string
	Command string
	Title   string
	Options []string
}

var pullConfig = []PullConfig{
	{
		Type:    "wank",
		Command: "/wank",
		Title:   "✈️今天打飞机了吗?",
		Options: []string{"Yes", "No"},
	},
	{
		Type:    "shit",
		Command: "/shit",
		Title:   "💩今天有拉屎了吗?",
		Options: []string{"Yes", "No"},
	},
}

func newPullHandler(config PullConfig) func(ctx context.Context, b *bot.Bot, update *models.Update) {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		logger := log.FromContext(ctx)
		logger.Info("newPullHandler",
			"cmd", config.Command,
			"type", config.Type,
		)
		date := time.Now().Format("2006-01-02")

		pull, exist, err := dao.GetPullByTypeAndDate(ctx, config.Type, date)
		if nil != err {
			logger.Error("GetPullByTypeAndDate error",
				"error", err)
			return
		}
		if exist {
			logger.Info("Pull already exists, forwarding message",
				"date", date,
				"type", config.Type,
				"messageID", pull.MessageID)

			// Forward the message if pull exists
			if pull.MessageID != 0 {
				// Get chat ID from the update
				chatID := update.Message.Chat.ID

				// Forward the existing message
				_, err = b.ForwardMessage(ctx, &bot.ForwardMessageParams{
					ChatID:     chatID,
					FromChatID: pull.ChatID,
					MessageID:  int(pull.MessageID),
				})

				if err != nil {
					logger.Error("Failed to forward message",
						"error", err)
				}
			}
			return
		}

		// Create new pull if it doesn't exist
		logger.Info("Creating new pull",
			"date", date,
			"type", config.Type)

		// Convert string options to InputPollOption objects
		var options []models.InputPollOption
		for _, option := range config.Options {
			options = append(options, models.InputPollOption{Text: option})
		}

		// Send a message first
		message, err := b.SendPoll(ctx, &bot.SendPollParams{
			ChatID:   update.Message.Chat.ID,
			Question: config.Title + " for " + date,
			Options:  options,
		})

		if err != nil {
			logger.Error("Failed to send message",
				"error", err)
			return
		}

		// Create and save the new pull
		newPull := dao.Pull{
			Type:      config.Type,
			Date:      date,
			MessageID: int64(message.ID),
			ChatID:    update.Message.Chat.ID,
		}

		err = dao.SavePull(ctx, newPull)
		if err != nil {
			logger.Error("Failed to save pull",
				"error", err)
			return
		}

		logger.Info("Successfully created new pull",
			"type", config.Type,
			"date", date,
			"messageID", message.ID)
	}
}
