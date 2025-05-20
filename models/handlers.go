package models

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func HandleStart(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	message := `欢迎使用饭堂记账机器人！

可用命令：
/add [+/-]金额 备注 - 添加一条支出或收入记录
/list - 查看所有记录
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
		}[tx.Type]

		sb.WriteString(fmt.Sprintf("%d. [%s] %.2f - %s（%s）by @%s\n", i+1, typeLabel, tx.Amount, tx.Note, tx.CreatedAt.Format("2006-01-02 15:04"), tx.UserDisplayName()))
	}

	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, sb.String()))
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
