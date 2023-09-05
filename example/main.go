package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yozel/telegramease"
)

// Global middleware
func authenticate(c *telegramease.Context) {
	c.Data["isAdmin"] = c.Update.FromChat() != nil && c.Update.FromChat().UserName == "yozel"
}

// Command middleware
func isAdmin(c *telegramease.Context, _ telegramease.Args) {
	if !c.Data["isAdmin"].(bool) {
		c.Reply("You are not an admin", "")
		c.Abort()
	}
}

// Command handler
func debug(c *telegramease.Context, _ telegramease.Args) {
	msg := ""
	incomingMsg, _ := c.GetMessage()
	b, err := json.MarshalIndent(incomingMsg, "", "  ")
	if err != nil {
		c.Reply(fmt.Errorf("error marshaling message: %w", err).Error(), "")
		c.Abort()
		return
	}
	msg = "```json\n" + string(b) + "\n```"
	c.Reply(msg, tgbotapi.ModeMarkdownV2)
}

// Command handler
func echo(c *telegramease.Context, args telegramease.Args) {
	msg := ""
	if len(args) > 0 {
		msg = args.GetAll()
	} else {
		msg = "You didn't provide any arguments"
	}
	c.Reply(msg, "")
}

func main() {
	bot, err := telegramease.NewBot("YOUR_TOKEN_HERE")
	if err != nil {
		log.Panic(err)
	}
	bot.Use(authenticate)
	bot.HandleCommand("debug", isAdmin, debug)
	bot.AddCommandHelper("debug", "", "Prints the message as JSON")

	bot.HandleCommand("echo", echo)
	bot.AddCommandHelper("echo", "<message>", "Echoes the message")

	if err := bot.Run(context.TODO()); err != nil {
		log.Printf("error running bot: %s", err)
	}
}
