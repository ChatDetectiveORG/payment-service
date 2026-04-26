package paymentservice

import (
	"os"
	"sync"

	e "github.com/ChatDetectiveORG/shared/errors"
	models "github.com/ChatDetectiveORG/shared/postgresModels"
	tele "gopkg.in/telebot.v4"
)

var (
	botOnce        sync.Once
	bot            *tele.Bot
	botErr         error
	mirrorBotMu    sync.Mutex
	mirrorBotCache = map[string]*tele.Bot{}
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

func GetBotByMirrorID(mirrorID string) (*tele.Bot, *e.ErrorInfo) {
	if mirrorID == "" {
		return GetBot()
	}
	mirrorBotMu.Lock()
	if bot, ok := mirrorBotCache[mirrorID]; ok {
		mirrorBotMu.Unlock()
		return bot, e.Nil()
	}
	mirrorBotMu.Unlock()

	id, err := models.ParseMirrorID(mirrorID)
	if e.IsNonNil(err) {
		return nil, err
	}
	mirror, err := models.FindMirrorByID(GetDB(), id)
	if e.IsNonNil(err) {
		return nil, err
	}
	token, err := mirror.DecryptToken(mirror.Owner)
	if e.IsNonNil(err) {
		return nil, err
	}
	mirrorBot, rawErr := tele.NewBot(tele.Settings{Token: token, Poller: nil})
	if rawErr != nil {
		return nil, e.FromError(rawErr, "failed to initialize mirror bot").WithSeverity(e.Notice)
	}

	mirrorBotMu.Lock()
	mirrorBotCache[mirrorID] = mirrorBot
	mirrorBotMu.Unlock()
	return mirrorBot, e.Nil()
}
