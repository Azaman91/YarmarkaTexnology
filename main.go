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

	if err := featureconnects.Createtable(context.Background(), conn); err != nil {
		log.Fatal("🚫 Не могу создать таблицу:", err)
	}

	fmt.Println("🚀 Сервер: http://localhost:8080")
	http.HandleFunc("/register", featureconnects.Connecthadler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
