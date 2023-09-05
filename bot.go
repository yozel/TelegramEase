package telegramease

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler func(ctx *Context)

type TelegramBot struct {
	Bot              *tgbotapi.BotAPI
	middleware       []Handler
	commands         map[string][]Handler
	callbacks        map[string][]Handler
	defaultHandler   Handler
	helpText         string
	commandsRegister []tgbotapi.BotCommand
}

func NewBot(token string) (*TelegramBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}
	bot.Debug = false
	b := &TelegramBot{
		Bot:              bot,
		middleware:       []Handler{},
		commands:         map[string][]Handler{},
		defaultHandler:   func(ctx *Context) {},
		commandsRegister: []tgbotapi.BotCommand{},
	}
	b.HandleDefault(b.helpHandler)
	return b, nil
}

func (b *TelegramBot) Use(handler Handler) {
	b.middleware = append(b.middleware, handler)
}

func (b *TelegramBot) helpHandler(ctx *Context) {
	ctx.Reply(b.helpText, "markdown")
	ctx.Abort()
}

func (b *TelegramBot) HandleCallback(callback string, handler ...Handler) {
	b.callbacks[callback] = handler
}

func (b *TelegramBot) HandleCommand(command string, handler ...Handler) {
	b.commands[command] = handler
}

func (b *TelegramBot) AddCommandHelper(command string, argdesc string, desc string) {
	b.helpText += "`/" + command + strings.TrimRight(" "+argdesc, " ") + "` - " + desc + "\n"
	b.commandsRegister = append(b.commandsRegister, tgbotapi.BotCommand{
		Command:     command,
		Description: desc,
	})
}

func (b *TelegramBot) HandleDefault(handler Handler) {
	b.defaultHandler = handler
}

func (b *TelegramBot) Run(ctx context.Context) error {
	b.HandleCommand("help", b.helpHandler)
	b.AddCommandHelper("help", "", "Help")
	_, err := b.Bot.Request(tgbotapi.NewSetMyCommands(b.commandsRegister...))
	if err != nil {
		return err
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.Bot.GetUpdatesChan(u)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			isCallback := false
			isEditedMsg := false
			msg := update.Message
			if msg == nil && update.EditedMessage != nil {
				isEditedMsg = true
				msg = update.EditedMessage
			} else if msg == nil && update.CallbackQuery != nil {
				isCallback = true
				msg = update.CallbackQuery.Message
			}
			b.handle(&Context{
				Bot:             b.Bot,
				Update:          update,
				Message:         msg,
				IsEditedMessage: isEditedMsg,
				IsCallback:      isCallback,
				Data:            make(map[string]interface{}),
			})
		}
	}
}

func (b *TelegramBot) handle(ctx *Context) {
	log.Printf("handling update %d", ctx.Update.UpdateID)
	for _, handler := range b.middleware {
		handler(ctx)
		if ctx.IsAborted {
			return
		}
	}
	if !ctx.IsCallback {
		if ctx.Message.Command() != "" {
			handlers, ok := b.commands[ctx.Message.Command()]
			if ok {
				for _, handler := range handlers {
					handler(ctx)
					if ctx.IsAborted {
						return
					}
				}
				return
			}
		}
		b.defaultHandler(ctx)
	} else {
		var v struct {
			CallbackName string `json:"callback_name"`
		}
		err := json.Unmarshal([]byte(ctx.Update.CallbackQuery.Data), &v)
		if err != nil {
			log.Printf("failed to unmarshal callback data: %v", err)
			return
		}
		handlers, ok := b.callbacks[v.CallbackName]
		if ok {
			for _, handler := range handlers {
				handler(ctx)
				if ctx.IsAborted {
					return
				}
			}
			return
		}
	}
}
