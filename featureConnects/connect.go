package featureconnects

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var db *pgxpool.Pool

func InitDB() error {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/YARMARKA_TEXNOLOGY")
	if err != nil {
		return fmt.Errorf("БД не запустилась: %w", err)
	}
	db = pool
	return nil
}

func Connecthadler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ctx := context.Background()

	username := r.Form.Get("username")
	email := r.Form.Get("email")
	password := r.Form.Get("password")
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
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"field":"hashpassword","message":"Неудалось хешировать данные"}`, 500)
		return
	}

	sqlQuery := `INSERT INTO users (name,email,password,created_at)
	VALUES ($1,$2,$3,$4);
	`
	_, err = db.Exec(ctx, sqlQuery, username, email, h, time.Now())
	if err != nil {
		http.Error(w, `{"field":"database","message":"Ошибка сохранение данных"}`, 500)
		fmt.Println("Error:", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, `{"redirect":"/dashboard"}`)
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
	return pgx.Connect(ctx, "postgres://postgres:pass@localhost:5432/YARMARKA_TEXNOLOGY")
}
