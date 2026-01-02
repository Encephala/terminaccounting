package main

import (
	"fmt"
	"terminaccounting/database"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	val, err := database.ParseCurrencyValue("5.50")
	fmt.Printf("%#v, %#v", val, err)

	val2, err := database.ParseCurrencyValue("-5.50")
	fmt.Printf("%#v, %#v", val2, err)
}
