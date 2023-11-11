package main

import (
	"database/sql"
	_ "database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"net/http"
	"os"
)

func handler(w http.ResponseWriter, r *http.Request) {

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"), os.Getenv("POSTGRES_PORT"), os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD"), os.Getenv("POSTGRES_DB"))
	//fmt.Fprint(w, psqlInfo)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		fmt.Fprint(w, "{\"Соединение установлено\": false}")
	}
	err = db.Ping()
	if err != nil {
		fmt.Fprint(w, "{\"Соединение установлено\": false}")
	} else {
		fmt.Fprint(w, "{\"Соединение установлено\": true}")
	}
	defer db.Close()
}

func main() {
	http.HandleFunc("/api/status", handler)
	http.ListenAndServe(":80", nil)
}
