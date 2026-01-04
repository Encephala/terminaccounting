package tat

import (
	"fmt"
	"terminaccounting/meta"
)

func (tw *TestWrapper) SwitchTab(direction meta.Sequence) *TestWrapper {
	tw.model, _ = tw.model.Update(meta.SwitchTabMsg{Direction: direction})

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

		tw.Send(meta.SwitchAppViewMsg{ViewType: viewType, Data: data})

	// TODO
	// case meta.TEXTMODALVIEWTYPE, meta.BANKIMPORTERVIEWTYPE:

	default:
		panic(fmt.Sprintf("unexpected meta.ViewType: %#v", viewType))
	}

	return tw
}
