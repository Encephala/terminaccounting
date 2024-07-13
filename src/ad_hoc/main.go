package main

import (
	"fmt"
	"terminaccounting/apps/entries"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db := sqlx.MustConnect("sqlite3", "test.db")

	rows := []entries.EntryRow{
		{
			Id:         0,
			Entry:      1,
			Ledger:     1,
			Account:    nil,
			Document:   nil,
			Value:      entries.DecimalValue{Whole: 6, Fractional: 9},
			Reconciled: false,
		},
		{
			Id:         0,
			Entry:      1,
			Ledger:     2,
			Account:    nil,
			Document:   nil,
			Value:      entries.DecimalValue{Whole: -6, Fractional: 9},
			Reconciled: false,
		},
	}

	fmt.Println("Inserting:", rows)
	// fmt.Println(entries.InsertRows(db, rows))

	a, err := entries.SelectRows(db)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%#+v\n", a)
}
