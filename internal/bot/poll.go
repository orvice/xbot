package bot

import (
	"context"
	"time"

	"butterfly.orx.me/core/log"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.orx.me/xbot/internal/dao"
)

var (
	boolFalse = false
)

type PollConfig struct {
	Type    string
	Command string
	Title   string
	Options []string
}

var pollConfig = []PollConfig{
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
	{
		Type:    "sex",
		Command: "/sex",
		Title:   "💕今天做爱了吗?",
		Options: []string{"Yes", "No"},
	},
	{
		Type:    "workout",
		Command: "/workout",
		Title:   "💪今天健身了吗?",
		Options: []string{"Yes", "No"},
	},
}

func newPollHandler(config PollConfig) func(ctx context.Context, b *bot.Bot, update *models.Update) {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		logger := log.FromContext(ctx).With("method", "PollHandler")
		logger.Info("newPollHandler",
			"cmd", config.Command,
			"type", config.Type,
		)
		date := time.Now().Format("2006-01-02")

		Poll, exist, err := dao.GetPollByTypeAndDate(ctx, config.Type, date)
		if nil != err {
			logger.Error("GetPollByTypeAndDate error",
				"error", err)
			return
		}
		if exist {
			logger.Info("Poll already exists, forwarding message",
				"date", date,
				"type", config.Type,
				"messageID", Poll.MessageID)

			// Forward the message if Poll exists
			if Poll.MessageID != 0 {
				// Get chat ID from the update
				chatID := update.Message.Chat.ID

				// Forward the existing message
				_, err = b.ForwardMessage(ctx, &bot.ForwardMessageParams{
					ChatID:     chatID,
					FromChatID: Poll.ChatID,
					MessageID:  int(Poll.MessageID),
				})

				if err != nil {
					logger.Error("Failed to forward message",
						"error", err)
				}
			}
			return
		}

		// Create new Poll if it doesn't exist
		logger.Info("Creating new Poll",
			"date", date,
			"type", config.Type)

		// Convert string options to InputPollOption objects
		var options []models.InputPollOption
		for _, option := range config.Options {
			options = append(options, models.InputPollOption{Text: option})
		}

		// Send a message first
		message, err := b.SendPoll(ctx, &bot.SendPollParams{
			ChatID:      update.Message.Chat.ID,
			Question:    config.Title + " for " + date,
			Options:     options,
			IsAnonymous: &boolFalse,
		})

		if err != nil {
			logger.Error("Failed to send message",
				"error", err)
			return
		}

		// Create and save the new Poll
		newPoll := dao.Poll{
			Type:      config.Type,
			Date:      date,
			MessageID: int64(message.ID),
			ChatID:    update.Message.Chat.ID,
			Poll:      message.Poll,
			PollID:    message.Poll.ID,
		}

		err = dao.SavePoll(ctx, newPoll)
		if err != nil {
			logger.Error("Failed to save Poll",
				"error", err)
			return
		}

		logger.Info("Successfully created new Poll",
			"type", config.Type,
			"date", date,
			"messageID", message.ID)
	}
}

func PollVoteHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	PollAnswer := update.PollAnswer
	logger := log.FromContext(ctx).With("method", "PollVoteHandler")
	logger.Info("PollVoteHandler",
		"pollAnswer", PollAnswer,
	)
}
