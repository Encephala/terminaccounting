package main

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"terminaccounting/database"
	"terminaccounting/meta"
	"terminaccounting/modals"

	overlay "github.com/Encephala/bubbletea-overlay"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type terminaccounting struct {
	appManager   *appManager
	modalManager *modals.ModalManager

	showModal             bool
	viewWidth, viewHeight int

	notifications       []notificationMsg
	displayNotification bool
	fatalError          error // To print to screen on exit

	// current vimesque input mode
	inputMode meta.InputMode
	// current motion
	currentMotion meta.Motion

	// vimesque command input
	commandInput           textinput.Model
	currentCommandIsSearch bool
}

func (ta *terminaccounting) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, ta.appManager.Init())

	err := database.UpdateCache()
	if err != nil {
		cmds = append(cmds, meta.MessageCmd(err))
	}

	return tea.Batch(cmds...)
}

func (ta *terminaccounting) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case meta.QuitMsg:
		if message.All {
			return ta, tea.Quit
		}

		if ta.showModal {
			ta.showModal = false

			return ta, nil
		}

		return ta, tea.Quit

	case error:
		slog.Debug(fmt.Sprintf("Error: %v", message))
		notification := notificationMsg{
			text:    message.Error(),
			isError: true,
		}
		ta.notifications = append(ta.notifications, notification)
		ta.displayNotification = true
		return ta, nil

	case tea.Cmd:
		return ta, message

	case tea.WindowSizeMsg:
		ta.viewWidth = message.Width
		ta.viewHeight = message.Height

		// -1 for status line
		// -1 for command line
		remainingHeight := message.Height - 1 - 1

		var cmd tea.Cmd
		ta.appManager, cmd = ta.appManager.Update(tea.WindowSizeMsg{
			Width:  message.Width,
			Height: remainingHeight,
		})

		return ta, cmd

	case meta.ShowTextMsg:
		ta.modalManager.Modal = modals.NewTextModal(message.Text)
		ta.showModal = true

		return ta, ta.modalManager.Init()

	case meta.ShowBankImporterMsg:
		ta.modalManager.Modal = modals.NewBankStatementImporter()
		ta.showModal = true

		return ta, ta.modalManager.Init()

	case meta.NotificationMessageMsg:
		notification := notificationMsg{
			text:    message.Message,
			isError: false,
		}
		ta.notifications = append(ta.notifications, notification)
		ta.displayNotification = true

		return ta, nil

	case meta.ShowNotificationsMsg:
		if len(ta.notifications) == 0 {
			return ta, meta.MessageCmd(errors.New("no messages to show"))
		}

		var rendered []string

		for i, notification := range ta.notifications {
			newLine := fmt.Sprintf("%d: %s", i, notification.String())

			rendered = append(rendered, newLine)
		}

		maxMessages := 20
		if len(rendered) > maxMessages {
			rendered = rendered[len(rendered)-maxMessages:]
		}

		return ta, meta.MessageCmd(meta.ShowTextMsg{Text: strings.Join(rendered, "\n")})

	case meta.FatalErrorMsg:
		slog.Error(fmt.Sprintf("Fatal error: %v", message.Error))
		ta.fatalError = message.Error
		return ta, tea.Quit

	case meta.SwitchModeMsg:
		return ta, ta.switchMode(message)

	case meta.ExecuteCommandMsg:
		command := ta.commandInput.Value()

		return ta.executeCommand(command)

	case meta.TryCompleteCommandMsg:
		commandSoFar := strings.Split(ta.commandInput.Value(), "")

		completed := ta.commandSet().Autocomplete(commandSoFar)

		if completed != nil {
			ta.commandInput.SetValue(strings.Join(completed, ""))
			ta.commandInput.CursorEnd()
		}

		return ta, nil

	case meta.SwitchViewMsg:
		// Always switch view in AppManager, can't switch view within a modal
		var cmd tea.Cmd
		ta.appManager, cmd = ta.appManager.Update(message)

		return ta, cmd

	case tea.KeyMsg:
		return ta.handleKeyMsg(message)

	case meta.RefreshCacheMsg:
		err := database.UpdateCache()
		if err != nil {
			return ta, meta.MessageCmd(err)
		}

		return ta, nil

	case meta.DebugPrintCacheMsg:
		slog.Debug(fmt.Sprintf("Ledgers: %#v", database.AvailableLedgers))
		slog.Debug(fmt.Sprintf("Accounts: %#v", database.AvailableAccounts))
		slog.Debug(fmt.Sprintf("Journals: %#v", database.AvailableJournals))

		return ta, nil
	}

	if ta.showModal {
		var cmd tea.Cmd
		ta.modalManager, cmd = ta.modalManager.Update(message)

		return ta, cmd
	}

	var cmd tea.Cmd
	ta.appManager, cmd = ta.appManager.Update(message)

	return ta, cmd
}

func (ta *terminaccounting) View() string {
	var result strings.Builder

	if ta.showModal {
		result.WriteString(newOverlay(ta).View())
	} else {
		result.WriteString(ta.appManager.View())
	}

	result.WriteString("\n")

	result.WriteString(ta.statusLineView())

	result.WriteString("\n")

	result.WriteString(ta.commandLineView())

	return result.String()
}

func (ta *terminaccounting) motionSet() *meta.CompleteMotionSet {
	var viewMotionSet meta.MotionSet

	if ta.showModal {
		viewMotionSet = ta.modalManager.CurrentMotionSet()
	} else {
		viewMotionSet = ta.appManager.CurrentMotionSet()
	}

	result := meta.NewCompleteMotionSet(viewMotionSet)

	return &result
}

func (ta *terminaccounting) commandSet() *meta.CompleteCommandSet {
	var viewCommandSet meta.CommandSet

	if ta.showModal {
		viewCommandSet = ta.modalManager.CurrentCommandSet()
	} else {
		viewCommandSet = ta.appManager.CurrentCommandSet()
	}

	result := meta.NewCompleteCommandSet(viewCommandSet)

	return &result
}

// Purely for readability
func (ta *terminaccounting) resetCurrentMotion() {
	ta.currentMotion = ta.currentMotion[:0]
}

func (ta *terminaccounting) switchMode(message meta.SwitchModeMsg) tea.Cmd {
	var cmd tea.Cmd

	if ta.inputMode == meta.COMMANDMODE {
		ta.commandInput.Reset()
		ta.commandInput.Blur()
	}

	ta.inputMode = message.InputMode

	if message.InputMode == meta.COMMANDMODE {
		ta.displayNotification = false
		ta.commandInput.Focus()

		isSearchMode := message.Data.(bool)

		if isSearchMode {
			ta.commandInput.Prompt = "/"
			ta.currentCommandIsSearch = true

			// If switching to search, send an empty search to views
			cmd = meta.MessageCmd(meta.UpdateSearchMsg{Query: ""})
		} else {
			ta.commandInput.Prompt = ":"
			ta.currentCommandIsSearch = false
		}
	}

	return cmd
}

func (ta *terminaccounting) executeCommand(command string) (*terminaccounting, tea.Cmd) {
	var cmd tea.Cmd

	if ta.currentCommandIsSearch {
		cmd = meta.MessageCmd(meta.UpdateSearchMsg{Query: command})
	} else if command != "" {
		command := strings.Split(command, "")

		if completion := ta.commandSet().Autocomplete(command); completion != nil {
			slog.Debug(fmt.Sprintf("Autocompleted command %q to %q",
				strings.Join(command, ""), strings.Join(completion, ""),
			))

			command = completion
		}

		commandMsg, ok := ta.commandSet().Get(command)
		if ok {
			cmd = meta.MessageCmd(commandMsg)
		} else {
			cmd = meta.MessageCmd(fmt.Errorf("invalid command: %q", strings.Join(command, "")))
		}
	}

	modeCmd := ta.switchMode(meta.SwitchModeMsg{InputMode: meta.NORMALMODE})

	return ta, tea.Batch(cmd, modeCmd)
}

func (ta *terminaccounting) handleKeyMsg(message tea.KeyMsg) (*terminaccounting, tea.Cmd) {
	if message.Type == tea.KeyCtrlC {
		return ta.handleCtrlC()
	}

	newMotion := append(ta.currentMotion, message.String())

	if completedMotionMsg, ok := ta.motionSet().Get(ta.inputMode, newMotion); ok {
		ta.resetCurrentMotion()

		return ta, meta.MessageCmd(completedMotionMsg)
	}

	if ta.motionSet().ContainsPath(ta.inputMode, newMotion) {
		ta.currentMotion = newMotion

		return ta, nil
	}

	// If this was the first button pressed/no keys before this in current motion,
	// forward the keyMsg to the appropriate input
	if len(ta.currentMotion) == 0 {
		switch ta.inputMode {
		case meta.NORMALMODE:
			// pass

		case meta.INSERTMODE:
			var cmd tea.Cmd
			if ta.showModal {
				ta.modalManager, cmd = ta.modalManager.Update(message)

				return ta, cmd
			}

			ta.appManager, cmd = ta.appManager.Update(message)

			return ta, cmd

		case meta.COMMANDMODE:
			var cmd tea.Cmd
			ta.commandInput, cmd = ta.commandInput.Update(message)

			if ta.currentCommandIsSearch {
				cmd = tea.Batch(cmd, meta.MessageCmd(meta.UpdateSearchMsg{Query: ta.commandInput.Value()}))
			}

			return ta, cmd

		default:
			panic(fmt.Sprintf("unexpected meta.InputMode: %#v", ta.inputMode))
		}
	}

	ta.resetCurrentMotion()

	return ta, meta.MessageCmd(fmt.Errorf("invalid motion: %q", newMotion.View()))
}

func (ta *terminaccounting) handleCtrlC() (*terminaccounting, tea.Cmd) {
	if len(ta.currentMotion) > 0 {
		ta.resetCurrentMotion()

		return ta, nil
	}

	switch ta.inputMode {
	case meta.NORMALMODE:
		return ta, nil

	case meta.INSERTMODE:
		return ta, meta.MessageCmd(meta.SwitchModeMsg{InputMode: meta.NORMALMODE})

	case meta.COMMANDMODE:
		modeCmd := meta.MessageCmd(meta.SwitchModeMsg{InputMode: meta.NORMALMODE})

		if ta.currentCommandIsSearch {
			searchCmd := meta.MessageCmd(meta.UpdateSearchMsg{Query: ""})

			return ta, tea.Batch(modeCmd, searchCmd)
		}

		return ta, modeCmd

	default:
		panic(fmt.Sprintf("unexpected meta.InputMode: %#v", ta.inputMode))
	}
}

type notificationMsg struct {
	text    string
	isError bool
}

func (nm notificationMsg) String() string {
	if nm.isError {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(nm.text)
	}

	return nm.text
}

func newOverlay(main *terminaccounting) *overlay.Model {
	return overlay.New(
		main.modalManager,
		main.appManager,
		overlay.Center,
		overlay.Center,
		0,
		1,
	)
}
