package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	featureconnects "study/featureConnects"
)

func main() {
	if err := featureconnects.InitDB(); err != nil {
		log.Fatal("🚫 БД:", err)
	}

	// Создаём таблицу (первый запуск)
	conn, err := featureconnects.Checkconnect(context.Background())
	if err != nil {
		log.Fatal("🚫 Подключение:", err)
	}
	defer conn.Close(context.Background())
	featureconnects.Createtable(context.Background(), conn)

	http.HandleFunc("/register", featureconnects.Connecthadler)
	http.HandleFunc("/verify", featureconnects.VerifiryCode)
	http.HandleFunc("/login", featureconnects.LoginHandler)
	http.HandleFunc("/dashboard", dashboardHandler)

	fmt.Println("🚀 http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "http://127.0.0.1:5500")
	fmt.Fprint(w, `
        <!DOCTYPE html>
        <html><head><title>Dashboard</title>
        <style>body{font-family:sans-serif;padding:2rem;background:#f0f8ff}</style></head>
        <body>
            <h1>🎉 Успешный вход!</h1>
            <p></p>
            <a href="http://127.0.0.1:5500/main.html">← Главная</a>
        </body></html>`)
}
