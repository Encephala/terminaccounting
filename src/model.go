package main

import (
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type model struct {
	db *sqlx.DB

	viewWidth, viewHeight int

	activeApp int
	apps      []meta.App

	displayedError error
	fatalError     error // To print to screen on exit

	// current vimesque input mode
	inputMode meta.InputMode
	// current motion
	currentMotion meta.Motion
	// known motionSet
	motionSet meta.CompleteMotionSet

	// vimesque command input
	commandInput textinput.Model
	// known commandSet
	commandSet meta.CompleteCommandSet
}
