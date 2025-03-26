package bot

import (
	"context"
	"fmt"
	"time"

	"butterfly.orx.me/core/log"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.orx.me/xbot/internal/dao"
)

var (
	boolFalse = false
)

const (
	pollTypeShit    = "shit"
	pollTypeWank    = "wank"
	pollTypeSex     = "sex"
	pollTypeWorkout = "workout"

	pullOptionYes = "Yes"
	pullOptionNo  = "No"
)

type PollConfig struct {
	Type    string
	Command string
	Title   string
	Options []string
}

var pollConfig = []PollConfig{
	{
		Type:    pollTypeWank,
		Command: "/wank",
		Title:   "✈️今天打飞机了吗?",
		Options: []string{pullOptionYes, pullOptionNo},
	},
	{
		Type:    pollTypeShit,
		Command: "/shit",
		Title:   "💩今天拉屎了吗?",
		Options: []string{pullOptionYes, pullOptionNo},
	},
	{
		Type:    pollTypeSex,
		Command: "/sex",
		Title:   "💕今天做爱了吗?",
		Options: []string{pullOptionYes, pullOptionNo},
	},
	{
		Type:    pollTypeWorkout,
		Command: "/workout",
		Title:   "💪今天健身了吗?",
		Options: []string{pullOptionYes, pullOptionNo},
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
	logger.Info("new poll answer",
		"update", update,
		"pollAnswer", PollAnswer,
	)

	poll, err := dao.GetPollByID(ctx, PollAnswer.PollID)
	if err != nil {
		logger.Error("GetPollByID error",
			"error", err)
		return
	}
	logger.Info("GetPollByID",
		"poll", poll,
	)

	userName := fmt.Sprintf("%s  %s%s", PollAnswer.User.Username, PollAnswer.User.FirstName, PollAnswer.User.LastName)

	var chatID int64 = poll.ChatID

	if PollAnswer.VoterChat != nil {
		chatID = PollAnswer.VoterChat.ID
	}

	if update.Message != nil {
		chatID = update.Message.Chat.ID
	}

	switch poll.Type {
	case pollTypeShit:
		logger.Info("new shit vote",
			"poll", poll,
			"PollAnswer", PollAnswer,
			"userName", userName,
			"voter.chat", PollAnswer.VoterChat,
		)

		if len(PollAnswer.OptionIDs) == 0 {

			if userName == "" {
				userName = "匿名用户"
			}

			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   fmt.Sprintf("🎉  %s 撤回了个拉屎投票.", userName),
			})
			if err != nil {
				logger.Error("Failed to send message", "error", err)
			}
			return
		}

		if PollAnswer.OptionIDs[0] == 0 { // Yes option
			logger.Info("new shit vote yes",
				"poll", poll,
				"user", PollAnswer.User,
				"PollAnswer", PollAnswer,
				"userName", userName,
			)

			if userName == "" {
				userName = "匿名用户"
			}

			resp, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: chatID,
				Text:   fmt.Sprintf("🎉 恭喜 %s 完成今日任务！💩\n祝您排便愉快，身体健康！", userName),
			})
			if err != nil {
				logger.Error("Failed to send congratulation message",
					"error", err)
				return
			}
			logger.Info("send success",
				"resp", resp,
			)
		}
	}
}
