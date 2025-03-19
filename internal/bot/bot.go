package bot

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
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
	b.RegisterHandler(bot.HandlerTypeMessageText, "/huahua", bot.MatchTypePrefix, huahuaHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/save_prompt", bot.MatchTypePrefix, savePromt)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/dns_query", bot.MatchTypePrefix, dnsQueryHandler)

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
		"update", update,
	)
	// save to store
	dao.SaveMessage(ctx, &dao.Message{
		Update: update,
	})
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

	start := time.Now()

	resp, err := openai.ChatCompletion(ctx, conf.Conf.OpenAI.Model, prompt.Promt, message)
	if nil != err {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Error",
		})
		return
	}

	duration := time.Since(start)
	logger.Info("ChatCompletion",
		"duration", duration,
		"resp", resp,
	)

	resp = fmt.Sprintf("Model: %s Duration: %s\n\n%s", conf.Conf.OpenAI.Model, duration, resp)

	sendResp, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   resp,
		ReplyParameters: &models.ReplyParameters{
			ChatID:                   update.Message.Chat.ID,
			MessageID:                update.Message.ID,
			AllowSendingWithoutReply: true,
			Quote:                    message,
		},
	})
	if nil != err {
		logger.Error("SendMessage error ",
			"error", err)
		return
	}
	logger.Info("SendMessage",
		"text", sendResp,
	)
}

func savePromt(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("savePromt",
		"text", update.Message.Text,
	)

	promt := update.Message.Text
	if strings.HasPrefix(promt, "/save_prompt") {
		promt = strings.TrimPrefix(promt, "/save_prompt")
	}

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
	if strings.HasPrefix(message, "/huahua ") {
		message = strings.TrimPrefix(message, "/huahua ")
	}

	resp, err := openai.GenImage(ctx, message)
	if nil != err {
		logger.Error("GenImage error ",
			"error", err)
		return
	}

	r, err := os.ReadFile(resp)
	if err != nil {
		logger.Error("ReadFile error ",
			"error", err)
		return
	}

	bf := strings.NewReader(string(r))

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
		return
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

	// get messages by chat id
	messages, err := dao.GetMessageByChatID(ctx, update.Message.Chat.ID)
	if nil != err {
		logger.Error("GetMessageByChatID error ",
			"error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Error retrieving messages. Please try again later.",
		})
		return
	}

	logger.Info("sumHandler", "len", len(messages))

	if len(messages) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "No messages found to summarize.",
		})
		return
	}

	// Build a conversation history from the messages
	var conversationBuilder strings.Builder
	conversationBuilder.WriteString("This is a conversation history from a Telegram chat. Please summarize the main topics discussed:\n\n")

	// Add up to the last 50 messages (to avoid token limits)
	maxMessages := 50
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

	// Get the prompt to use
	summarizationPrompt := "You are a helpful assistant that summarizes conversations. Provide a concise summary of the key points discussed in this conversation. Focus on the main topics, questions asked, and decisions made."

	start := time.Now()

	// Call OpenAI to summarize the conversation
	conversationText := conversationBuilder.String()
	summary, err := openai.ChatCompletion(ctx, conf.Conf.OpenAI.Model, summarizationPrompt, conversationText)
	if err != nil {
		logger.Error("ChatCompletion error", "error", err)
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Error generating summary. Please try again later.",
		})
		return
	}

	duration := time.Since(start)
	logger.Info("Summary generated",
		"duration", duration,
		"chars", len(summary),
	)

	// Format the response
	response := fmt.Sprintf("📝 **Chat Summary**\n\nModel: %s\nProcessed %d messages in %s\n\n%s", 
		conf.Conf.OpenAI.Model, 
		len(messages), 
		duration.Round(time.Millisecond), 
		summary)

	// Edit the loading message with the summary
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
		// If no loading message was sent, send a new message with the summary
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   response,
		})
	}
}
