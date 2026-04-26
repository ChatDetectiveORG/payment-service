package paymentservice

import (
	"os"
	"sync"

	e "github.com/ChatDetectiveORG/shared/errors"
	tele "gopkg.in/telebot.v4"
)

var (
	botOnce sync.Once
	bot     *tele.Bot
	botErr  error
)

func GetBot() (*tele.Bot, *e.ErrorInfo) {
	botOnce.Do(func() {
		token := os.Getenv("TELEGRAM_BOT_TOKEN")
		if token == "" {
			botErr = e.NewError("missing env TELEGRAM_BOT_TOKEN", "telegram bot token is not configured")
			return
		}
		bot, botErr = tele.NewBot(tele.Settings{Token: token, Poller: nil})
	})
	if botErr != nil {
		return nil, e.FromError(botErr, "failed to initialize telegram bot").WithSeverity(e.Critical)
	}
	return bot, e.Nil()
}
