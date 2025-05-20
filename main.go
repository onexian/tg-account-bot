package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"log"
	"os"
	"tg-account-bot/models"
)

func main() {
	godotenv.Load()

	// Init DB
	if err := models.InitDB(); err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	// Init Telegram Bot
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Fatal("Bot 初始化失败:", err)
	}

	bot.Debug = true
	log.Printf("已登录: %s", bot.Self.UserName)

	// 只调用一次
	models.HandleSetCommands(bot)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil || !update.Message.IsCommand() {
			continue
		}

		switch update.Message.Command() {
		case "start":
			models.HandleStart(bot, update.Message)
		case "add":
			models.HandleRecord(bot, update.Message)
		case "list":
			models.HandleList(bot, update.Message)
		case "balance":
			models.HandleBalance(bot, update.Message)
		case "summary":
			models.HandleSummary(bot, update.Message)
		case "week":
			models.HandleWeek(bot, update.Message)
		case "month":
			models.HandleMonth(bot, update.Message)

		}
	}
}
