package bot

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"butterfly.orx.me/core/log"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.orx.me/xbot/internal/conf"
	"go.orx.me/xbot/internal/dao"
	"go.orx.me/xbot/internal/pkg/openai"
)

var (
	defaultBot *bot.Bot
)

func Init() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
	}
	b, err := bot.New(conf.Conf.TelegramBotToken, opts...)
	if nil != err {
		return err
	}

	defaultBot = b
	b.RegisterHandler(bot.HandlerTypeMessageText, "/hello", bot.MatchTypePrefix, helloHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/gpt", bot.MatchTypePrefix, gptHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "gpt", bot.MatchTypePrefix, gptHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/sum", bot.MatchTypePrefix, sumHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/ask", bot.MatchTypePrefix, askHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/huahua", bot.MatchTypePrefix, huahuaHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/save_prompt", bot.MatchTypePrefix, savePromt)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/dns_query", bot.MatchTypePrefix, dnsQueryHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/getid", bot.MatchTypeExact, getIDHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/me", bot.MatchTypeExact, meHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/hualao", bot.MatchTypeExact, hualaoHandler)

	for _, config := range pollConfig {
		b.RegisterHandler(bot.HandlerTypeMessageText, config.Command, bot.MatchTypePrefix, newPollHandler(config))
	}

	resp, err := b.SetWebhook(ctx, &bot.SetWebhookParams{
		URL: fmt.Sprintf("%s/v1/webhook", conf.Conf.Host),
	})
	if err != nil {
		slog.Error("set webhook error",
			"error", err)
	}
	slog.Info("set webhook success", "resp", resp)

	go b.StartWebhook(context.Background())
	return nil
}

func GetBot() *bot.Bot {
	return defaultBot
}

func helloHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("helloHandler",
		"update", update,
	)
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Hello, *" + bot.EscapeMarkdown(update.Message.From.FirstName) + "*",
		ParseMode: models.ParseModeMarkdown,
	})
	if nil != err {
		logger.Error("SendMessage error ",
			"error", err)
	}
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("defaultHandler",
		"update", update.ID,
	)

	if update.PollAnswer != nil {
		PollVoteHandler(ctx, b, update)
	}

	// save to store
	err := dao.GetMessageStorage().SaveMessage(ctx, &dao.Message{
		Update: update,
	})
	if nil != err {
		logger.Error("SaveMessage error ",
			"error", err)
	}
}

func gptHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)

	prompt, err := dao.GetPromt(ctx, update.Message.Chat.ID)
	if err != nil {
		logger.Error("GetPromt error ",
			"error", err)
	}
	if prompt.Promt == "" {
		prompt.Promt = "You are a helpful assistant."
	}

	message := update.Message.Text

	// remove gpt or /gpt prefix
	if strings.HasPrefix(message, "/gpt ") {
		message = strings.TrimPrefix(message, "/gpt ")
	} else if strings.HasPrefix(message, "gpt ") {
		message = strings.TrimPrefix(message, "gpt ")
	}

	logger.Info("gptHandler",
		"prompt", prompt.Promt,
		"message", message,
	)

	// Send a processing message first
	loadingMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Processing your request...",
		ReplyParameters: &models.ReplyParameters{
			ChatID:                   update.Message.Chat.ID,
			MessageID:                update.Message.ID,
			AllowSendingWithoutReply: true,
			Quote:                    message,
		},
	})
	if err != nil {
		logger.Error("Failed to send loading message", "error", err)
	}

	start := time.Now()

	resp, err := openai.ChatCompletion(ctx, conf.Conf.OpenAI.Model, prompt.Promt, message)
	if nil != err {
		if loadingMsg != nil {
			// Update the loading message with the error
			_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.Message.Chat.ID,
				MessageID: loadingMsg.ID,
				Text:      "Error processing your request. Please try again.",
			})
			if err != nil {
				logger.Error("Failed to edit message", "error", err)
				// If editing fails, send a new message
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Error processing your request. Please try again.",
				})
			}
		} else {
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Error processing your request. Please try again.",
			})
		}
		return
	}

	duration := time.Since(start)
	logger.Info("ChatCompletion",
		"duration", duration,
		"resp", resp,
	)

	formattedResp := fmt.Sprintf("Model: %s Duration: %s\n\n%s", conf.Conf.OpenAI.Model, duration, resp)

	if loadingMsg != nil {
		// Update the loading message with the response
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: loadingMsg.ID,
			Text:      formattedResp,
		})
		if err != nil {
			logger.Error("Failed to edit message", "error", err)
			// If editing fails, send a new message
			sendResp, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   formattedResp,
				ReplyParameters: &models.ReplyParameters{
					ChatID:                   update.Message.Chat.ID,
					MessageID:                update.Message.ID,
					AllowSendingWithoutReply: true,
					Quote:                    message,
				},
			})
			if err != nil {
				logger.Error("SendMessage error ", "error", err)
				return
			}
			logger.Info("SendMessage", "text", sendResp)
		}
	} else {
		// If no loading message was sent, send a new message with the response
		sendResp, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   formattedResp,
			ReplyParameters: &models.ReplyParameters{
				ChatID:                   update.Message.Chat.ID,
				MessageID:                update.Message.ID,
				AllowSendingWithoutReply: true,
				Quote:                    message,
			},
		})
		if err != nil {
			logger.Error("SendMessage error ", "error", err)
			return
		}
		logger.Info("SendMessage", "text", sendResp)
	}
}

func savePromt(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("savePromt",
		"text", update.Message.Text,
	)

	promt := update.Message.Text
	// Directly use TrimPrefix without conditional check
	promt = strings.TrimPrefix(promt, "/save_prompt")

	err := dao.SavePromt(ctx, dao.Promt{
		ChatID:    update.Message.Chat.ID,
		Promt:     promt,
		CreatedAt: time.Now().Unix(),
	})
	if nil != err {
		logger.Error("SavePromt error ",
			"error", err)
		return
	}

	// send message
	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Prompt saved",
	})
	if nil != err {
		logger.Error("SendMessage error ",
			"error", err)
		return
	}
}

func huahuaHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("huahuaHandler",
		"text", update.Message.Text,
	)

	message := update.Message.Text
	// Directly use TrimPrefix without conditional check
	message = strings.TrimPrefix(message, "/huahua ")

	// Send a processing message first
	loadingMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Generating image...",
		ReplyParameters: &models.ReplyParameters{
			ChatID:                   update.Message.Chat.ID,
			MessageID:                update.Message.ID,
			AllowSendingWithoutReply: true,
			Quote:                    message,
		},
	})
	if err != nil {
		logger.Error("Failed to send loading message", "error", err)
	}

	imageData, err := openai.GenImage(ctx, message)
	if nil != err {
		logger.Error("GenImage error ",
			"error", err)

		if loadingMsg != nil {
			// Update the loading message with the error
			_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.Message.Chat.ID,
				MessageID: loadingMsg.ID,
				Text:      "Error generating image. Please try again.",
			})
			if err != nil {
				logger.Error("Failed to edit message", "error", err)
			}
		}
		return
	}

	// Extract the base64 data from the imageData string
	// Format is: data:image/jpeg;base64,<actual-base64-data>
	parts := strings.Split(imageData, ",")
	if len(parts) != 2 {
		logger.Error("Invalid image data format")

		if loadingMsg != nil {
			// Update the loading message with the error
			_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.Message.Chat.ID,
				MessageID: loadingMsg.ID,
				Text:      "Error processing image data. Please try again.",
			})
			if err != nil {
				logger.Error("Failed to edit message", "error", err)
			}
		}
		return
	}

	// Decode the base64 data
	imgData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		logger.Error("Base64 decode error", "error", err)

		if loadingMsg != nil {
			// Update the loading message with the error
			_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    update.Message.Chat.ID,
				MessageID: loadingMsg.ID,
				Text:      "Error decoding image data. Please try again.",
			})
			if err != nil {
				logger.Error("Failed to edit message", "error", err)
			}
		}
		return
	}

	bf := bytes.NewReader(imgData)

	// If we have a loading message, delete it before sending the photo
	if loadingMsg != nil {
		_, err = b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: loadingMsg.ID,
		})
		if err != nil {
			logger.Error("Failed to delete loading message", "error", err)
		}
	}

	params := &bot.SendPhotoParams{
		ChatID: update.Message.Chat.ID,
		Photo: &models.InputFileUpload{
			Filename: "huahua.png",
			Data:     bf,
		},
		Caption: message,
		ReplyParameters: &models.ReplyParameters{
			ChatID:                   update.Message.Chat.ID,
			MessageID:                update.Message.ID,
			AllowSendingWithoutReply: true,
			Quote:                    message,
		},
	}

	// send image
	_, err = b.SendPhoto(ctx, params)
	if nil != err {
		logger.Error("SendPhoto error ",
			"error", err)

		// If we failed to send the photo and already deleted the loading message,
		// send a new error message
		_, err = b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Error sending the generated image. Please try again.",
			ReplyParameters: &models.ReplyParameters{
				ChatID:                   update.Message.Chat.ID,
				MessageID:                update.Message.ID,
				AllowSendingWithoutReply: true,
			},
		})
		if err != nil {
			logger.Error("Failed to send error message", "error", err)
		}
		return
	}
}

// prepareChatHistory prepares conversation history from messages
func prepareChatHistory(messages []*dao.Message, maxMessages int, prefix string) string {
	var conversationBuilder strings.Builder
	conversationBuilder.WriteString(prefix)

	// Add up to the last N messages (to avoid token limits)
	startIdx := 0
	if len(messages) > maxMessages {
		startIdx = len(messages) - maxMessages
	}

	for i := startIdx; i < len(messages); i++ {
		if messages[i].Update.Message != nil && messages[i].Update.Message.Text != "" {
			name := "User"
			if messages[i].Update.Message.From != nil {
				if messages[i].Update.Message.From.Username != "" {
					name = "@" + messages[i].Update.Message.From.Username
				} else if messages[i].Update.Message.From.FirstName != "" {
					name = messages[i].Update.Message.From.FirstName
					if messages[i].Update.Message.From.LastName != "" {
						name += " " + messages[i].Update.Message.From.LastName
					}
				}
			}
			conversationBuilder.WriteString(fmt.Sprintf("%s: %s\n", name, messages[i].Update.Message.Text))
		}
	}

	return conversationBuilder.String()
}

// processChatHistory handles the common logic for processing chat history with OpenAI
func processChatHistory(ctx context.Context, b *bot.Bot, update *models.Update, loadingMsg *models.Message,
	prompt string, messagePrefix string, responseTitle string, noMessagesText string) {

	logger := log.FromContext(ctx)

	// get messages by chat id
	messages, err := dao.GetMessageStorage().GetMessageByChatID(ctx, update.Message.Chat.ID)
	if nil != err {
		logger.Error("GetMessageByChatID error ",
			"error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Error retrieving messages. Please try again later.",
		})
		return
	}

	logger.Info("Chat history processing", "len", len(messages))

	if len(messages) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   noMessagesText,
		})
		return
	}

	// Build a conversation history from the messages
	conversationText := prepareChatHistory(messages, 50, messagePrefix)

	start := time.Now()

	// Use models defined in config with fallback
	models := conf.Conf.SummaryModels
	if len(models) == 0 {
		// Fallback to the default model if no models defined in config
		models = []string{conf.Conf.OpenAI.Model}
	}

	// Call OpenAI to process the conversation with multiple model options
	result, usedModel, err := openai.ChatCompletionWithModels(ctx, models, prompt, conversationText)
	if err != nil {
		logger.Error("ChatCompletion error", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Error processing chat history. Please try again later.",
		})
		return
	}

	duration := time.Since(start)
	logger.Info("AI response generated",
		"duration", duration,
		"model", usedModel,
		"chars", len(result),
	)

	// Format the response
	response := fmt.Sprintf("%s\n\nModel: %s\nProcessed %d messages in %s\n\n%s",
		responseTitle,
		usedModel,
		len(messages),
		duration.Round(time.Millisecond),
		result)

	// Edit the loading message with the result
	if loadingMsg != nil {
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: loadingMsg.ID,
			Text:      response,
		})
		if err != nil {
			logger.Error("Failed to edit message", "error", err)
			// If editing fails, send a new message
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   response,
			})
		}
	} else {
		// If no loading message was sent, send a new message with the result
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   response,
		})
	}
}

func sumHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("sumHandler",
		"text", update.Message.Text,
	)

	// Send a loading message to the user
	loadingMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Summarizing chat messages...",
	})
	if err != nil {
		logger.Error("Failed to send loading message", "error", err)
	}

	// Get the prompt to use
	summarizationPrompt := "你是一个帮助用户总结对话的助手。请提供这个对话中讨论的关键点的简明摘要。重点关注主要话题、提出的问题以及做出的决定。"
	messagePrefix := "这是一个Telegram聊天历史记录。请总结讨论的主要话题：\n\n"

	processChatHistory(
		ctx,
		b,
		update,
		loadingMsg,
		summarizationPrompt,
		messagePrefix,
		"📝 **Chat Summary**",
		"No messages found to summarize.",
	)
}

func askHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("askHandler",
		"text", update.Message.Text,
	)

	// Extract the question from user input
	userMessage := update.Message.Text
	userQuestion := ""
	if strings.HasPrefix(userMessage, "/ask ") {
		userQuestion = strings.TrimPrefix(userMessage, "/ask ")
	}

	// If no question was provided, inform the user
	if userQuestion == "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please provide a question after the /ask command. For example: /ask What did we decide about the project deadline?",
		})
		return
	}

	// Send a loading message to the user
	loadingMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Searching chat history for an answer...",
		ReplyParameters: &models.ReplyParameters{
			ChatID:                   update.Message.Chat.ID,
			MessageID:                update.Message.ID,
			AllowSendingWithoutReply: true,
			Quote:                    userQuestion,
		},
	})
	if err != nil {
		logger.Error("Failed to send loading message", "error", err)
	}

	// Get messages by chat id
	messages, err := dao.GetMessageStorage().GetMessageByChatID(ctx, update.Message.Chat.ID)
	if nil != err {
		logger.Error("GetMessageByChatID error ",
			"error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Error retrieving messages. Please try again later.",
		})
		return
	}

	logger.Info("Chat history processing", "len", len(messages))

	if len(messages) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "No chat history found to answer your question.",
		})
		return
	}

	// Create a customized prompt that includes the user's question
	answerPrompt := fmt.Sprintf("你是一个帮助用户从对话历史中找答案的助手。请根据提供的聊天记录，回答用户的问题：'%s'。如果聊天记录中没有足够的信息来回答这个问题，请诚实地说明，并提供一些基于现有信息的建议或见解。", userQuestion)
	messagePrefix := "这是一个Telegram聊天历史记录：\n\n"

	// Build a conversation history from the messages
	conversationText := prepareChatHistory(messages, 50, messagePrefix)

	start := time.Now()

	// Use models defined in config with fallback
	models := conf.Conf.SummaryModels
	if len(models) == 0 {
		// Fallback to the default model if no models defined in config
		models = []string{conf.Conf.OpenAI.Model}
	}

	// Call OpenAI to process the conversation with multiple model options
	result, usedModel, err := openai.ChatCompletionWithModels(ctx, models, answerPrompt, conversationText)
	if err != nil {
		logger.Error("ChatCompletion error", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Error processing your question. Please try again later.",
		})
		return
	}

	duration := time.Since(start)
	logger.Info("AI response generated",
		"duration", duration,
		"model", usedModel,
		"chars", len(result),
	)

	// Format the response
	response := fmt.Sprintf("❓ **Answer to: %s**\n\nModel: %s\nProcessed in %s\n\n%s",
		userQuestion,
		usedModel,
		duration.Round(time.Millisecond),
		result)

	// Edit the loading message with the result
	if loadingMsg != nil {
		_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: loadingMsg.ID,
			Text:      response,
		})
		if err != nil {
			logger.Error("Failed to edit message", "error", err)
			// If editing fails, send a new message
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   response,
			})
		}
	} else {
		// If no loading message was sent, send a new message with the result
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   response,
		})
	}
}

func getIDHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("getIDHandler",
		"chat_id", update.Message.Chat.ID,
	)

	text := fmt.Sprintf("Chat ID: `%d`", update.Message.Chat.ID)

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      text,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		logger.Error("SendMessage error", "error", err)
	}
}

func meHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("meHandler",
		"from", update.Message.From,
	)

	user := update.Message.From
	var info strings.Builder

	info.WriteString("User Information:\n")
	info.WriteString(fmt.Sprintf("ID: `%d`\n", user.ID))
	info.WriteString(fmt.Sprintf("First Name: *%s*\n", bot.EscapeMarkdown(user.FirstName)))

	if user.LastName != "" {
		info.WriteString(fmt.Sprintf("Last Name: *%s*\n", bot.EscapeMarkdown(user.LastName)))
	}
	if user.Username != "" {
		info.WriteString(fmt.Sprintf("Username: @%s\n", user.Username))
	}
	if user.LanguageCode != "" {
		info.WriteString(fmt.Sprintf("Language: `%s`\n", user.LanguageCode))
	}
	if user.IsBot {
		info.WriteString("Type: Bot\n")
	} else {
		info.WriteString("Type: User\n")
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      info.String(),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		logger.Error("SendMessage error", "error", err)
	}
}

type userStats struct {
	FirstName string
	LastName  string
	Username  string
	Count     int
}

func hualaoHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx).With("method", "hualaoHandler")
	logger.Info("new hualao req",
		"chat_id", update.Message.Chat.ID,
	)

	// Send initial loading message
	loadingMsg, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Generating chat statistics for the last 7 days...",
	})
	if err != nil {
		logger.Error("Failed to send loading message", "error", err)
		return
	}

	// Get messages from the last 7 days
	messages, err := dao.GetMessageStorage().GetMessageByChatID(ctx, update.Message.Chat.ID)
	if err != nil {
		logger.Error("Failed to get messages", "error", err)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: loadingMsg.ID,
			Text:      "Error: Failed to retrieve chat messages.",
		})
		return
	}

	if len(messages) == 0 {
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.Message.Chat.ID,
			MessageID: loadingMsg.ID,
			Text:      "No messages found in the last 7 days.",
		})
		return
	}

	// Create a map to store user statistics
	stats := make(map[int64]*userStats)

	// Count messages for each user
	for _, msg := range messages {
		if msg.Update.Message == nil || msg.Update.Message.From == nil {
			continue
		}

		from := msg.Update.Message.From
		userID := from.ID

		if _, exists := stats[userID]; !exists {
			stats[userID] = &userStats{
				FirstName: from.FirstName,
				LastName:  from.LastName,
				Username:  from.Username,
				Count:     0,
			}
		}
		stats[userID].Count++
	}

	// Convert map to slice for sorting
	rankings := make([]struct {
		UserID int64
		Stats  *struct {
			FirstName string
			LastName  string
			Username  string
			Count     int
		}
	}, 0, len(stats))

	for userID, userStats := range stats {
		rankings = append(rankings, struct {
			UserID int64
			Stats  *struct {
				FirstName string
				LastName  string
				Username  string
				Count     int
			}
		}{
			UserID: userID,
			Stats: &struct {
				FirstName string
				LastName  string
				Username  string
				Count     int
			}{
				FirstName: userStats.FirstName,
				LastName:  userStats.LastName,
				Username:  userStats.Username,
				Count:     userStats.Count,
			},
		})
	}

	// Sort by message count (descending)
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Stats.Count > rankings[j].Stats.Count
	})

	// Build the response message
	var response strings.Builder
	response.WriteString("*Chat Activity Leaderboard \\(Last 7 Days\\)*\n\n")
	response.WriteString(fmt.Sprintf("Total Messages: *%d*\n\n", len(messages)))

	// Add rankings
	for i, rank := range rankings {
		if i >= 10 { // Only show top 10
			break
		}

		stats := rank.Stats
		name := stats.FirstName
		if stats.LastName != "" {
			name += " " + stats.LastName
		}
		name = bot.EscapeMarkdown(name)

		username := ""
		if stats.Username != "" {
			username = fmt.Sprintf(" \\(@%s\\)", stats.Username)
		}

		medal := ""
		switch i {
		case 0:
			medal = "🥇"
		case 1:
			medal = "🥈"
		case 2:
			medal = "🥉"
		default:
			medal = "👤"
		}

		response.WriteString(fmt.Sprintf("%d\\. %s *%s*%s \\- %d messages\n",
			i+1, medal, name, username, stats.Count))
	}

	// Add footer with timestamp
	response.WriteString(fmt.Sprintf("\n_Generated at: %s_",
		bot.EscapeMarkdown(time.Now().Format("2006-01-02 15:04:05"))))

	// Update the loading message with the results
	_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    update.Message.Chat.ID,
		MessageID: loadingMsg.ID,
		Text:      response.String(),
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		logger.Error("Failed to update message with results",
			"error", err)
		return
	}
}
