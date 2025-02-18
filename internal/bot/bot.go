package bot

import (
	"context"
	"os"
	"os/signal"

	"butterfly.orx.me/core/log"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"go.orx.me/xbot/internal/conf"
	"go.orx.me/xbot/internal/pkg/openai"
)

func Init() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
	}
	b, err := bot.New(os.Getenv(conf.Conf.TelegramBotToken), opts...)
	if nil != err {
		return err
	}
	b.RegisterHandler(bot.HandlerTypeMessageText, "/hello", bot.MatchTypeExact, helloHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/gpt", bot.MatchTypeExact, gptHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "gpt", bot.MatchTypeExact, gptHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/save_prompt", bot.MatchTypeExact, savePromt)
	go b.Start(ctx)
	return nil
}

func helloHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Hello, *" + bot.EscapeMarkdown(update.Message.From.FirstName) + "*",
		ParseMode: models.ParseModeMarkdown,
	})
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Say /hello",
	})
}

func gptHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("gptHandler",
		"text", update.Message.Text,
	)
	resp, err := openai.ChatCompletion(ctx, update.Message.Text)
	if nil != err {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Error",
		})
		return
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   resp,
	})
}

func savePromt(ctx context.Context, b *bot.Bot, update *models.Update) {
	logger := log.FromContext(ctx)
	logger.Info("savePromt",
		"text", update.Message.Text,
	)
}
