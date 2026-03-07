package featureconnects

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
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
	if r.Method == "GET" {
		http.ServeFile(w, r, "register.html")
		return
	}
	if r.Method != "POST" {
		http.Error(w, `{"error":"Только POST"}`, 405)
		return
	}
	body, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))

	if err := r.ParseForm(); err != nil {
		http.Error(w, `{"error":"Ошибка парсинга формы"}`, 400)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")

	fmt.Println("🟢 Запрос:", username)
	fmt.Println("🔍 Email:", email)
	fmt.Println("🔍 Пароль:", password)

	// ✅ Проверка пустых полей
	if username == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"username","message":"Ник пустой"}`, 400)
		return
	}
	if email == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"email","message":"Email пустой"}`, 400)
		return
	}
	if password == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"password","message":"Пароль пустой"}`, 400)
		return
	}

	ctx := context.Background()

	if exists(ctx, db, username) {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"username","message":"Ник занят"}`, 400)
		return
	}
	if exists(ctx, db, email) {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"email","message":"Почта занята"}`, 400)
		return
	}

	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("❌ Bcrypt error:", err)
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"password","message":"Ошибка хеширования"}`, 500)
		return
	}

	sqlQuery := `INSERT INTO users (name, email, password, created_at) VALUES ($1, $2, $3, $4)`
	_, err = db.Exec(ctx, sqlQuery, username, email, h, time.Now())
	if err != nil {
		fmt.Println("❌ DB Error:", err)
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"database","message":"Ошибка сохранения данных"}`, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, `{"success":true,"redirect":"/dashboard","message":"✅ Регистрация успешна!"}`)
}

func exists(ctx context.Context, pool *pgxpool.Pool, name string) bool {
	var exists bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE name = $1)`, name).Scan(&exists)
	return exists == true && err == nil
}

func Createtable(ctx context.Context, conn *pgx.Conn) error {
	sqlQuery := `
	CREATE TABLE users(
		id SERIAL PRIMARY KEY,
		name VARCHAR(50) UNIQUE NOT NULL,
		email VARCHAR(100) UNIQUE NOT NULL,
		password TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT NOW()
	);`

	_, err := conn.Exec(ctx, sqlQuery)
	if err != nil {
		return err
	}

	return nil
}

func Checkconnect(ctx context.Context) (*pgx.Conn, error) {
	return pgx.Connect(ctx, "postgres://postgres:postgres@localhost:5432/YARMARKA_TEXNOLOGY?sslmode=disable")
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5500")
	if r.Method == "GET" {
		http.ServeFile(w, r, "register.html")
		return
	}
	if r.Method != "POST" {
		http.Error(w, `{"error":"Только POST"}`, 405)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Ошибка чтения запроса"}`, 400)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err := r.ParseForm(); err != nil {
		http.Error(w, `{"error":"Ошибка парсинга формы"}`, 400)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	fmt.Println("🟢 Запрос:", username)
	fmt.Println("🔍 Пароль:", password)

	if username == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"username","message":"Ник пустой"}`, 400)
		return
	}
	if password == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"password","message":"Пароль пустой"}`, 400)
		return
	}
	ctx := context.Background()
	e := checkLogin(ctx, db, username, password)
	if e != nil {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Неверный логин или пароль"}`, 401)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"success":true,"redirect":"/dashboard","message":"Добро пожаловать!"}`)
}

func checkLogin(ctx context.Context, pool *pgxpool.Pool, user, password string) error {
	sqlQuery := `
	SELECT name, password 
	FROM users 
	WHERE name = $1;
	`
	var dbName string
	var dbPassword []byte
	err := pool.QueryRow(ctx, sqlQuery, user).Scan(&dbName, &dbPassword)
	if err == sql.ErrNoRows {
		return fmt.Errorf("пользователь не найден")
	}
	if err != nil {
		return fmt.Errorf("ошибка БД: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword(dbPassword, []byte(password)); err != nil {
		return fmt.Errorf("неверный пароль")
	}
	return nil
}
