package models

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleSetCommands(bot *tgbotapi.BotAPI) {
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "开始使用"},
		{Command: "add", Description: "添加记录"},
		{Command: "list", Description: "查看记录"},
		{Command: "clear", Description: "结余历史"},
		{Command: "balance", Description: "查看余额"},
		{Command: "summary", Description: "查看总收支"},
		{Command: "week", Description: "查看本周支出"},
		{Command: "month", Description: "查看本月支出"},
	}

	cfg := tgbotapi.NewSetMyCommands(commands...)

	if _, err := bot.Request(cfg); err != nil {
		log.Printf("设置命令菜单失败: %v", err)
	} else {
		log.Println("命令菜单设置成功！")
	}
}

func HandleStart(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	message := `欢迎使用饭堂记账机器人！

可用命令：
/add [+/-]金额 备注 - 添加一条支出或收入记录
/list - 查看所有记录
/clear - 结余历史
/balance - 查看总余额
/summary - 每人总收支统计
/week [last] - 本周或上周支出总额
/month [last] - 本月或上月支出总额`

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, message))
}

func HandleRecord(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 2 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "格式应为：类型 金额 备注\n如：/add [+/-]金额 备注"))
		return
	}

	amountStr := args[0]
	note := ""
	if len(args) > 1 {
		note = strings.Join(args[1:], " ")
	}

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "金额格式错误，请输入数字（可带正负号）"))
		return
	}

	txType := "income"
	if amount < 0 {
		txType = "expense"
		amount = -amount // 存入数据库时存为正数
	}

	EnsureUserExists(msg.From)

	tx := Transaction{
		UserID: int64(msg.From.ID),
		Type:   txType,
		Amount: amount,
		Note:   note,
	}

	if err := InsertTransaction(tx); err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "记录失败: "+err.Error()))
		return
	}

	reply := fmt.Sprintf("✅ 已记录：%s %.2f，备注：%s",
		map[string]string{"income": "收入", "expense": "支出"}[txType],
		amount,
		note,
	)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, reply))
}

func HandleList(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	txs, err := GetLatestTransactions(20)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "获取记录失败"))
		return
	}

	if len(txs) == 0 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "暂无记录"))
		return
	}

	var sb strings.Builder
	for i, tx := range txs {
		typeLabel := map[string]string{
			"income":  "收入",
			"expense": "支出",
			"clear":   "结余",
		}[tx.Type]

		sb.WriteString(fmt.Sprintf("%d. [%s] %.2f - %s（%s）by @%s\n", i+1, typeLabel, tx.Amount, tx.Note, tx.CreatedAt.Format("2006-01-02 15:04"), tx.UserDisplayName()))
	}

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, sb.String()))
}

func HandleClear(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {

	tgUserID := int64(msg.From.ID)
	adminUIDsStr := os.Getenv("TELEGRAM_ADMIN_UID")
	if adminUIDsStr != "" {
		adminList := strings.Split(adminUIDsStr, ",")
		isAdmin := false
		for _, uidStr := range adminList {
			uidStr = strings.TrimSpace(uidStr)
			if uidStr == "" {
				continue
			}
			uid, err := strconv.ParseInt(uidStr, 10, 64)
			if err != nil {
				continue // 跳过无效 UID
			}
			if tgUserID == uid {
				isAdmin = true
				break
			}
		}
		if !isAdmin {
			bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "❌ 非管理员账号，禁止使用该命令"))
			return
		}
	}

	// 查询最后一条记录
	txs, err := GetLatestTransactions(1)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "⚠️ 查询记录失败: "+err.Error()))
		return
	}

	if len(txs) > 0 && txs[0].Type == "clear" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ 上一条已为 clear，无需重复记录"))
		return
	}

	tx := Transaction{
		UserID: tgUserID,
		Type:   "clear",
		Amount: 0,
		Note:   "结余清空",
	}

	if err := InsertTransaction(tx); err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "清空失败: "+err.Error()))
		return
	}

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ 结余成功，历史数据不再统计"))
}

func HandleBalance(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	income, expense, err := CalculateTotalBalance()
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "获取失败"))
		return
	}
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID,
		fmt.Sprintf("📊 总余额：%.2f\n收入：%.2f\n支出：%.2f", income-expense, income, expense)))
}

func HandleSummary(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	summary, err := GetUserSummary()
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "统计失败"))
		return
	}

	var sb strings.Builder
	sb.WriteString("👥 每人统计：\n")
	for _, item := range summary {
		sb.WriteString(fmt.Sprintf("%s：收入 %.2f，支出 %.2f，净额 %.2f\n",
			item.UserName, item.Income, item.Expense, item.Income-item.Expense))
	}
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, sb.String()))
}

func HandleWeek(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	arg := strings.TrimSpace(msg.CommandArguments())

	isLast := arg == "last"
	title := "本周支出总额"
	if isLast {
		title = "上周支出总额"
	}

	total, err := GetWeeklyExpense(isLast)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "查询失败："+err.Error()))
		return
	}

	resp := fmt.Sprintf("%s：%.2f", title, total)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, resp))

}

func HandleMonth(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	arg := strings.TrimSpace(msg.CommandArguments())

	isLast := arg == "last"
	title := "本月支出总额"
	if isLast {
		title = "上月支出总额"
	}

	total, err := GetMonthlyExpense(isLast)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "查询失败："+err.Error()))
		return
	}

	resp := fmt.Sprintf("%s：%.2f", title, total)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, resp))

}
