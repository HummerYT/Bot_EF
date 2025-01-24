package main

import (
	"Telegram-Bot/pkg/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
)

func main() {
	bot, err := tgbotapi.NewBotAPI("")
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true //флаг для того, чтоб в консоли постоянно выводились логи с обновлениями и тд в боте

	telegram.InitDB("")

	telegrambot := telegram.NewBot(bot)
	if err := telegrambot.Start(); err != nil {
		log.Fatal(err)
	}
}
