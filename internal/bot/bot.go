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
	b.RegisterHandler(bot.HandlerTypeMessageText, "/huahua", bot.MatchTypePrefix, huahuaHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/save_prompt", bot.MatchTypePrefix, savePromt)

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

	resp, err := openai.ChatCompletion(ctx, prompt.Promt, message)
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
	if strings.HasPrefix(promt, "/save_prompt ") {
		promt = strings.TrimPrefix(promt, "/save_prompt ")
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
