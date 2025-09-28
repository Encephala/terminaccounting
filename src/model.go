package main

import (
	"fmt"
	"log/slog"
	"strings"
	"terminaccounting/meta"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type terminaccounting struct {
	appManager *appManager

	viewWidth, viewHeight int

	notifications       []notificationMsg
	displayNotification bool
	fatalError          error // To print to screen on exit

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

func (ta *terminaccounting) Init() tea.Cmd {
	return ta.appManager.Init()
}

func (ta *terminaccounting) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.QuitMsg:
		return ta, tea.Quit

	case tea.Cmd:
		return ta, message

	case tea.WindowSizeMsg:
		ta.viewWidth = message.Width
		ta.viewHeight = message.Height

		// -1 for status line
		remainingHeight := message.Height - 1
		newAppManager, cmd := ta.appManager.Update(tea.WindowSizeMsg{
			Width:  message.Width,
			Height: remainingHeight,
		})
		ta.appManager = newAppManager.(*appManager)

		return ta, cmd

	case error:
		slog.Debug(fmt.Sprintf("Error: %v", message))
		notification := notificationMsg{
			text:    message.Error(),
			isError: true,
		}
		ta.notifications = append(ta.notifications, notification)
		ta.displayNotification = true
		return ta, nil

	case meta.NotificationMessageMsg:
		notification := notificationMsg{
			text:    message.Message,
			isError: false,
		}
		ta.notifications = append(ta.notifications, notification)
		ta.displayNotification = true

		return ta, nil

	case meta.FatalErrorMsg:
		slog.Error(fmt.Sprintf("Fatal error: %v", message.Error))
		ta.fatalError = message.Error
		return ta, tea.Quit

	case meta.UpdateViewMotionSetMsg:
		ta.motionSet.ViewMotionSet = message

		return ta, nil

	case meta.UpdateViewCommandSetMsg:
		ta.commandSet.ViewCommandSet = message

		return ta, nil

	case meta.SwitchModeMsg:
		ta.switchMode(message.InputMode)

		return ta, nil

	case meta.ExecuteCommandMsg:
		command := ta.commandInput.Value()

		return ta.executeCommand(command)

	case tea.KeyMsg:
		return ta.handleKeyMsg(message)
	}

	newAppManager, cmd := ta.appManager.Update(message)
	ta.appManager = newAppManager.(*appManager)

	return ta, cmd
}

func (ta *terminaccounting) View() string {
	var result strings.Builder

	result.WriteString(ta.appManager.View())

	result.WriteString("\n")

	result.WriteString(ta.statusLineView())

	return result.String()
}

func (ta *terminaccounting) resetCurrentMotion() {
	ta.currentMotion = ta.currentMotion[:0]
}

func (ta *terminaccounting) switchMode(newMode meta.InputMode) {
	if ta.inputMode == meta.COMMANDMODE {
		ta.commandInput.Reset()
		ta.commandInput.Blur()
	}

	ta.inputMode = newMode

	if newMode == meta.COMMANDMODE {
		ta.displayNotification = false
		ta.commandInput.Focus()
	}
}

func (ta *terminaccounting) executeCommand(command string) (*terminaccounting, tea.Cmd) {
	commandMsg, ok := ta.commandSet.Get(strings.Split(command, ""))

	ta.switchMode(meta.NORMALMODE)

	if !ok {
		return ta, meta.MessageCmd(fmt.Errorf("invalid command: %v", ta.commandInput.Value()))
	}

	return ta, meta.MessageCmd(commandMsg)
}

func (ta *terminaccounting) handleKeyMsg(message tea.KeyMsg) (*terminaccounting, tea.Cmd) {
	// ctrl+c to reset the current motion can't be handled as a motion itself,
	// because then for instance ["g", "ctrl+c"] would be recognised as an invalid motion
	if ta.inputMode == meta.NORMALMODE && message.Type == tea.KeyCtrlC {
		ta.resetCurrentMotion()

		return ta, nil
	}

	ta.displayNotification = false
	ta.currentMotion = append(ta.currentMotion, message.String())

	if !ta.motionSet.ContainsPath(ta.inputMode, ta.currentMotion) {
		switch ta.inputMode {
		case meta.NORMALMODE:
			cmd := meta.MessageCmd(fmt.Errorf("invalid motion: %s", ta.currentMotion.View()))

			ta.resetCurrentMotion()

			return ta, cmd

		// In INSERT and COMMAND mode, a key stroke that isn't a motion but gets sent to the appropriate input
		case meta.INSERTMODE:
			newAppManager, cmd := ta.appManager.Update(message)
			ta.appManager = newAppManager.(*appManager)

			ta.resetCurrentMotion()

			return ta, cmd

		case meta.COMMANDMODE:
			var cmd tea.Cmd
			ta.commandInput, cmd = ta.commandInput.Update(message)

			ta.resetCurrentMotion()

			return ta, cmd
		}
	}

	completedMotionMsg, ok := ta.motionSet.Get(ta.inputMode, ta.currentMotion)
	if !ok {
		// The currentMotion is the start of an existing motion, wait for more inputs
		return ta, nil
	}

	ta.resetCurrentMotion()

	return ta, meta.MessageCmd(completedMotionMsg)
}

type notificationMsg struct {
	text    string
	isError bool
}
