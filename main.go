package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	featureconnects "study/featureConnects"
)

func main() {
	// Подключаемся к БД
	if err := featureconnects.InitDB(); err != nil {
		log.Fatal("🚫 БД недоступна:", err)
	}

	// Создаём таблицу
	conn, err := featureconnects.Checkconnect(context.Background())
	if err != nil {
		log.Fatal("🚫 Не могу подключиться:", err)
	}
	defer conn.Close(context.Background())

	fmt.Println("🚀 Сервер: http://localhost:8080")
	http.HandleFunc("/register", featureconnects.Connecthadler)
	http.HandleFunc("/login", featureconnects.LoginHandler)
	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5500")
		fmt.Fprint(w, `
            <!DOCTYPE html>
            <html>
            <head><title>Dashboard</title>
            <style>body{font-family:sans-serif;padding:2rem;background:#f0f8ff}</style>
            </head>
            <body>
                <h1>Успешный вход!</h1>
                <p></p>
                <a href="/register">← Назад</a>
            </body></html>`)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
