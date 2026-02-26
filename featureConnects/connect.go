package featureconnects

import (
	"context"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func Connecthadler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	username := r.Form.Get("username")
	email := r.Form.Get("email")
	password := r.Form.Get("password")

	if exists(username) {
		w.Header().Set("Connect-Type", "application/json")
		http.Error(w, `{"field":"username","message":"Ник занят"}`, 400)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"redirect":"/dashboard"}`)
}

func exists(name string) bool {

}

func Createtable(ctx context.Context, conn *pgx.Conn) error {
	sqlQuery := `
	CREATE TABLE tasks(
		id SERIAL PRIMARY KEY,
		name VARCHAR(200) NOT NULL,
		email VARCHAR(200) NOT NULL,
		password VARCHAR(200) NOT NULL
	);`

	_, err := conn.Exec(ctx, sqlQuery)
	if err != nil {
		return err
	}

	return nil
}

func Checkconnect(ctx context.Context) (*pgx.Conn, error) {
	return pgx.Connect(ctx, "postgres://postgres:pass@localhost:5432/YARMARKA TEXNOLOGY")
}
