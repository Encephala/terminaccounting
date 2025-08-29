package main

import (
	"fmt"
	"terminaccounting/database"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db := sqlx.MustConnect("sqlite3", "test.db")
	database.DB = db

	val, err := database.ParseCurrencyValue("5.50")
	fmt.Printf("%#v, %#v", val, err)

	val2, err := database.ParseCurrencyValue("-5.50")
	fmt.Printf("%#v, %#v", val2, err)
}
