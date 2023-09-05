package telegramease

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Context struct {
	Bot       *tgbotapi.BotAPI
	Update    tgbotapi.Update
	Data      map[string]interface{}
	IsAborted bool
}

func (ctx *Context) GetMessage() (*tgbotapi.Message, error) {
	if ctx.Update.Message != nil {
		return ctx.Update.Message, nil
	} else if ctx.Update.EditedMessage != nil {
		return ctx.Update.EditedMessage, nil
	} else {
		return nil, fmt.Errorf("no supported message in update: %+v", ctx.Update)
	}
}

func (c *Context) Abort() {
	c.IsAborted = true
}

func (c *Context) Reply(text string, parseMode string) error {
	fromChat := c.Update.FromChat()
	if fromChat == nil {
		return fmt.Errorf("no chat to reply to")
	}
	msg := tgbotapi.NewMessage(fromChat.ID, text)
	msg.ParseMode = parseMode
	if _, err := c.Bot.Send(msg); err != nil {
		log.Printf("failed to send message: %v", err)
		return err
	}
	return nil
}
