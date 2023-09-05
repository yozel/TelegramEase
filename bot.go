package telegramease

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Handler func(ctx *Context)
type CmdHandler func(ctx *Context, args Args)
type CallbackHandler func(ctx *Context, data string)

type TelegramBot struct {
	Bot              *tgbotapi.BotAPI
	middleware       []Handler
	commands         map[string][]CmdHandler
	callbacks        map[string][]CallbackHandler
	defaultHandler   CmdHandler
	helpText         string
	commandsRegister []tgbotapi.BotCommand
}

type Args []string

func (a Args) GetAll() string {
	return strings.Join(a, " ")
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
		commands:         map[string][]CmdHandler{},
		callbacks:        map[string][]CallbackHandler{},
		defaultHandler:   func(ctx *Context, _ Args) {},
		commandsRegister: []tgbotapi.BotCommand{},
	}
	b.HandleDefault(b.helpHandler)
	return b, nil
}

func (b *TelegramBot) Use(handler Handler) {
	b.middleware = append(b.middleware, handler)
}

func (b *TelegramBot) helpHandler(ctx *Context, _ Args) {
	ctx.Reply(b.helpText, "markdown")
	ctx.Abort()
}

func (b *TelegramBot) HandleCallback(callback string, handler ...CallbackHandler) {
	b.callbacks[callback] = handler
}

func (b *TelegramBot) HandleCommand(command string, handler ...CmdHandler) {
	b.commands[command] = handler
}

func (b *TelegramBot) AddCommandHelper(command string, argdesc string, desc string) {
	b.helpText += "`/" + command + strings.TrimRight(" "+argdesc, " ") + "` - " + desc + "\n"
	b.commandsRegister = append(b.commandsRegister, tgbotapi.BotCommand{
		Command:     command,
		Description: desc,
	})
}

func (b *TelegramBot) HandleDefault(handler CmdHandler) {
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
			b.handle(&Context{
				Bot:    b.Bot,
				Update: update,
				Data:   make(map[string]interface{}),
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
	msg, err := ctx.GetMessage()
	if err == nil {
		if msg.Command() != "" {
			args := []string{}
			for _, arg := range strings.Split(msg.CommandArguments(), " ") {
				if arg != "" {
					args = append(args, arg)
				}
			}
			handlers, ok := b.commands[msg.Command()]
			if ok {
				for _, handler := range handlers {
					handler(ctx, args)
					if ctx.IsAborted {
						return
					}
				}
				return
			}
		}
		b.defaultHandler(ctx, []string{})
	} else if ctx.Update.CallbackQuery != nil {
		args := strings.SplitN(ctx.Update.CallbackQuery.Data, ":", 2)
		if len(args) != 2 {
			log.Printf("invalid callback data: %s", ctx.Update.CallbackQuery.Data)
			return
		}
		handlers, ok := b.callbacks[args[0]]
		if ok {
			for _, handler := range handlers {
				handler(ctx, args[1])
				if ctx.IsAborted {
					return
				}
			}
			return
		}
	} else {
		log.Printf("unsupported update type: %+v", ctx.Update)
	}
}
