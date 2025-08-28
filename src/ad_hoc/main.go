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

	rows := []database.EntryRow{
		{
			Id:         0,
			Entry:      1,
			Ledger:     1,
			Account:    nil,
			Document:   nil,
			Value:      database.CurrencyValue(609),
			Reconciled: false,
		},
		{
			Id:         0,
			Entry:      1,
			Ledger:     2,
			Account:    nil,
			Document:   nil,
			Value:      database.CurrencyValue(-609),
			Reconciled: false,
		},
	}

	fmt.Println("Inserting:", rows)
	// fmt.Println(entries.InsertRows(db, rows))

	a, err := database.SelectRows()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%#+v\n", a)
}
