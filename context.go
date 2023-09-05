package telegramease

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Context struct {
	Bot             *tgbotapi.BotAPI
	Message         *tgbotapi.Message
	IsEditedMessage bool
	IsCallback      bool
	Update          tgbotapi.Update
	Data            map[string]interface{}
	IsAborted       bool
}

func (c *Context) Abort() {
	c.IsAborted = true
}

func (c *Context) Reply(text string, parseMode string) error {
	msg := tgbotapi.NewMessage(c.Message.Chat.ID, text)
	msg.ParseMode = parseMode
	if _, err := c.Bot.Send(msg); err != nil {
		log.Printf("failed to send message: %v", err)
		return err
	}
	return nil
}
