package models

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var db *sql.DB

type Transaction struct {
	ID        int64
	UserID    int64
	Type      string
	Amount    float64
	Note      string
	CreatedAt time.Time
	UserName  string
}

type UserSummary struct {
	UserName string
	Income   float64
	Expense  float64
}

func InitDB() error {
	user := os.Getenv("MYSQL_USER")
	pass := os.Getenv("MYSQL_PASSWORD")
	host := os.Getenv("MYSQL_HOST")
	dbname := os.Getenv("MYSQL_DATABASE")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, pass, host, dbname)

	var err error
	db, err = sql.Open("mysql", dsn)
	return err
}

func EnsureUserExists(user *tgbotapi.User) error {
	_, err := db.Exec(`INSERT IGNORE INTO users (id, username, first_name, last_name) VALUES (?, ?, ?, ?)`,
		user.ID, user.UserName, user.FirstName, user.LastName)
	return err
}

func InsertTransaction(t Transaction) error {
	_, err := db.Exec(`INSERT INTO transactions (user_id, type, amount, note) VALUES (?, ?, ?, ?)`,
		t.UserID, t.Type, t.Amount, t.Note)
	return err
}

func GetLatestTransactions(limit int) ([]Transaction, error) {
	rows, err := db.Query(`
        SELECT t.id, t.user_id, t.type, t.amount, t.note, t.created_at,
               COALESCE(u.username, u.first_name, '') AS name
        FROM transactions t
        LEFT JOIN users u ON t.user_id = u.id
        ORDER BY t.created_at DESC
        LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var tx Transaction
		err := rows.Scan(&tx.ID, &tx.UserID, &tx.Type, &tx.Amount, &tx.Note, &tx.CreatedAt, &tx.UserName)
		if err != nil {
			return nil, err
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func (t *Transaction) UserDisplayName() string {
	if t.UserName != "" {
		return t.UserName
	}
	return fmt.Sprintf("用户 %d", t.UserID)
}

func CalculateTotalBalance() (income, expense float64, err error) {
	err = db.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'income'`).Scan(&income)
	if err != nil {
		return
	}
	err = db.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM transactions WHERE type = 'expense'`).Scan(&expense)
	return
}

func GetUserSummary() ([]UserSummary, error) {
	rows, err := db.Query(`
        SELECT COALESCE(u.username, u.first_name, '未知') AS name,
               SUM(CASE WHEN t.type='income' THEN t.amount ELSE 0 END) AS income,
               SUM(CASE WHEN t.type='expense' THEN t.amount ELSE 0 END) AS expense
        FROM transactions t
        LEFT JOIN users u ON t.user_id = u.id
        GROUP BY t.user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []UserSummary
	for rows.Next() {
		var s UserSummary
		if err := rows.Scan(&s.UserName, &s.Income, &s.Expense); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, nil
}

func GetWeeklyExpense(isLast bool) (float64, error) {
	var total float64
	var query string

	if isLast {
		query = `
			SELECT IFNULL(SUM(amount), 0)
			FROM transactions
			WHERE type = 'expense'
			  AND YEARWEEK(DATE(created_at), 1) = YEARWEEK(CURDATE() - INTERVAL 1 WEEK, 1)
		`
	} else {
		query = `
			SELECT IFNULL(SUM(amount), 0)
			FROM transactions
			WHERE type = 'expense'
			  AND YEARWEEK(DATE(created_at), 1) = YEARWEEK(CURDATE(), 1)
		`
	}

	err := db.QueryRow(query).Scan(&total)
	return total, err
}

func GetMonthlyExpense(isLast bool) (float64, error) {
	var total float64
	var query string

	if isLast {
		query = `
			SELECT IFNULL(SUM(amount), 0)
			FROM transactions
			WHERE type = 'expense'
			  AND DATE_FORMAT(created_at, '%Y-%m') = DATE_FORMAT(CURDATE() - INTERVAL 1 MONTH, '%Y-%m')
		`
	} else {
		query = `
			SELECT IFNULL(SUM(amount), 0)
			FROM transactions
			WHERE type = 'expense'
			  AND DATE_FORMAT(created_at, '%Y-%m') = DATE_FORMAT(CURDATE(), '%Y-%m')
		`
	}

	err := db.QueryRow(query).Scan(&total)
	return total, err
}
