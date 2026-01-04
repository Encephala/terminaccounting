package tat

import (
	"fmt"
	"terminaccounting/meta"
)

func (tw *TestWrapper) SwitchMode(mode meta.InputMode, data ...any) *TestWrapper {
	switch mode {
	case meta.INSERTMODE, meta.NORMALMODE:
		if len(data) != 0 {
			panic("mode doesn't take an argument")
		}

		tw.Send(meta.SwitchModeMsg{InputMode: mode})

	case meta.COMMANDMODE:
		if len(data) != 1 {
			panic("wrong data format passed")
		}

		tw.Send(meta.SwitchModeMsg{InputMode: mode, Data: data[0]})

	default:
		panic(fmt.Sprintf("unexpected meta.InputMode: %#v", mode))
	}

	return tw
}

func (tw *TestWrapper) SwitchTab(direction meta.Sequence) *TestWrapper {
	tw.Send(meta.SwitchTabMsg{Direction: direction})

	return tw
}

func (tw *TestWrapper) SwitchView(viewType meta.ViewType, data ...any) *TestWrapper {
	switch viewType {
	case meta.LISTVIEWTYPE, meta.CREATEVIEWTYPE:
		if data != nil {
			panic("view doesn't take argument")
		}

		tw.Send(meta.SwitchAppViewMsg{ViewType: viewType})

	case meta.DETAILVIEWTYPE, meta.UPDATEVIEWTYPE, meta.DELETEVIEWTYPE:
		if len(data) != 1 {
			panic("wrong data format passed")
		}

		tw.Send(meta.SwitchAppViewMsg{ViewType: viewType, Data: data[0]})

	// TODO
	// case meta.TEXTMODALVIEWTYPE, meta.BANKIMPORTERVIEWTYPE:

	default:
		panic(fmt.Sprintf("unexpected meta.ViewType: %#v", viewType))
	}

	return tw
}
