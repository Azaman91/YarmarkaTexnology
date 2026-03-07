package featureconnects

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/smtp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var db *pgxpool.Pool

func InitDB() error {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://postgres:postgres@localhost:5432/YARMARKA_TEXNOLOGY?sslmode=disable")
	if err != nil {
		return fmt.Errorf("БД не запустилась: %w", err)
	}
	db = pool
	return nil
}

func Connecthadler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5500")

	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Только POST"}`, 405)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, `{"error":"Ошибка чтения"}`, 400)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	if err := r.ParseForm(); err != nil {
		jsonError(w, `{"error":"Ошибка парсинга"}`, 400)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	if username == "" || email == "" || password == "" {
		jsonError(w, `{"error":"Заполните все поля"}`, 400)
		return
	}

	ctx := context.Background()

	var isVerified bool

	if err := db.QueryRow(ctx, `SELECT is_verified FROM users WHERE name = $1`, username).Scan(&isVerified); err == nil && !isVerified {
		db.Exec(ctx, `DELETE FROM users WHERE name = $1`, username)
	}

	if err := db.QueryRow(ctx, `SELECT is_verified FROM users WHERE email = $1`, email).Scan(&isVerified); err == nil && !isVerified {
		db.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)
	}

	code, err := generatePassword()
	if err != nil {
		jsonError(w, `{"error":"Ошибка генерации кода"}`, 500)
		return
	}

	h, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	_, err = db.Exec(ctx,
		`INSERT INTO users (name, email, password, verification_code, is_verified, created_at) 
         VALUES ($1, $2, $3, $4, false, $5)`,
		username, email, h, code, time.Now())

	if err != nil {
		fmt.Println("❌ DB Error:", err)
		jsonError(w, `{"error":"Ошибка БД"}`, 500)
		return
	}

	go sendEmail(email, code)

	successJSON(w, fmt.Sprintf(`{"success":true,"message":"✅ Код отправлен на %s!"}`, email))
}

func VerifiryCode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5500")

	if r.Method != "POST" {
		jsonError(w, `{"error":"Только POST"}`, 405)
		return
	}

	r.ParseForm()
	code := r.FormValue("code")
	email := r.FormValue("email")

	if len(code) != 6 {
		jsonError(w, `{"field":"code","message":"Код 6 цифр"}`, 400)
		return
	}

	ctx := context.Background()
	var userId int
	var dbCode string

	err := db.QueryRow(ctx,
		`SELECT id, verification_code FROM users WHERE email = $1 AND is_verified = false`,
		email).Scan(&userId, &dbCode)

	if err != nil || code != dbCode {
		jsonError(w, `{"field":"code","message":"Неверный код"}`, 400)
		return
	}

	_, err = db.Exec(ctx,
		`UPDATE users SET is_verified = true, verification_code = NULL WHERE id = $1`,
		userId)

	if err != nil {
		jsonError(w, `{"error":"Ошибка БД"}`, 500)
		return
	}

	successJSON(w, `{"success":true,"message":"✅ Email подтверждён!"}`)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5500")

	if r.Method != "POST" {
		jsonError(w, `{"error":"Только POST"}`, 405)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		jsonError(w, `{"error":"Ошибка чтения"}`, 400)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	if err := r.ParseForm(); err != nil {
		jsonError(w, `{"error":"Ошибка парсинга"}`, 400)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		jsonError(w, `{"error":"Заполните поля"}`, 400)
		return
	}

	ctx := context.Background()
	if err := checkLogin(ctx, db, username, password); err != nil {
		jsonError(w, `{"error":"Неверный логин/пароль или email не подтверждён"}`, 401)
		return
	}

	successJSON(w, `{"success":true,"redirect":"http://localhost:8080/dashboard","message":"Добро пожаловать!"}`)
}

func checkLogin(ctx context.Context, pool *pgxpool.Pool, user, password string) error {
	var dbName string
	var dbPassword []byte
	var isVerified bool

	err := pool.QueryRow(ctx,
		`SELECT name, password, is_verified FROM users WHERE name = $1`,
		user).Scan(&dbName, &dbPassword, &isVerified)

	if err == sql.ErrNoRows {
		return fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return fmt.Errorf("ошибка БД")
	}

	if !isVerified {
		return fmt.Errorf("email не подтверждён")
	}

	if err := bcrypt.CompareHashAndPassword(dbPassword, []byte(password)); err != nil {
		return fmt.Errorf("неверный пароль")
	}
	return nil
}

func exists(ctx context.Context, pool *pgxpool.Pool, name string) bool {
	var isVerified bool
	err := pool.QueryRow(ctx,
		`SELECT is_verified FROM users WHERE name = $1`,
		name).Scan(&isVerified)
	if err == sql.ErrNoRows || !isVerified {
		return false
	}

	return true
}

func emailExists(ctx context.Context, pool *pgxpool.Pool, email string) bool {
	var isVerified bool
	err := pool.QueryRow(ctx,
		`SELECT is_verified FROM users WHERE email = $1`,
		email).Scan(&isVerified)

	if err == sql.ErrNoRows {
		return false
	}

	if err != nil {
		return false
	}

	return isVerified
}

func generatePassword() (string, error) {
	code := make([]byte, 6)
	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code[i] = byte('0' + num.Int64())
	}
	return string(code), nil
}

func sendEmail(email, code string) error {
	from := "artemprudnikov23@gmail.com"
	pass := "wpwgxiqkfrywzrci"

	// ✅ ПРАВИЛЬНЫЙ формат письма
	msg := []byte(fmt.Sprintf(`From: %s <%s>
To: %s
Subject: Подтверждение регистрации YARMARKA_TEXNOLOGY
MIME-Version: 1.0
Content-Type: text/html; charset=UTF-8

<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"></head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
    <h2 style="color: #667eea;">🎉 Добро пожаловать!</h2>
    <p>Ваш код подтверждения:</p>
    <h1 style="background: #667eea; color: white; padding: 20px; text-align: center; 
                font-size: 2.5em; letter-spacing: 10px; margin: 20px 0;">%s</h1>
    <p><strong>Действителен 10 минут</strong></p>
    <hr style="border: none; height: 1px; background: #eee;">
    <p style="color: #666; font-size: 0.9em;">YARMARKA_TEXNOLOGY Auth Service</p>
</body>
</html>`,
		"YARMARKA_TEXNOLOGY", from, email, code))

	// ✅ TLS + Gmail SMTP
	auth := smtp.PlainAuth("", from, pass, "smtp.gmail.com")

	// Подключение с TLS
	return smtp.SendMail("smtp.gmail.com:587", auth, from, []string{email}, msg)
}

// Вспомогательные функции
func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, msg, code)
}

func successJSON(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, msg)
}

func Createtable(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS users(
            id SERIAL PRIMARY KEY,
            name VARCHAR(50) UNIQUE NOT NULL,
            email VARCHAR(100) UNIQUE NOT NULL,
            password TEXT NOT NULL,
            verification_code VARCHAR(6),
            is_verified BOOLEAN DEFAULT false,
            created_at TIMESTAMP DEFAULT NOW()
        )`)
	return err
}

func Checkconnect(ctx context.Context) (*pgx.Conn, error) {
	return pgx.Connect(ctx, "postgres://postgres:postgres@localhost:5432/YARMARKA_TEXNOLOGY?sslmode=disable")
}
