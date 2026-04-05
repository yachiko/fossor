package common

import "github.com/charmbracelet/bubbles/key"

// MainKeyMap defines key bindings for the main screen.
type MainKeyMap struct {
	Quit             key.Binding
	Enter            key.Binding
	Pull             key.Binding
	PullAll          key.Binding
	Fetch            key.Binding
	FetchAll         key.Binding
	Search           key.Binding
	Escape           key.Binding
	Filter           key.Binding
	SwitchDefault    key.Binding
	SwitchDefaultAll key.Binding
	Sort1            key.Binding
	Sort2            key.Binding
	Sort3            key.Binding
	Sort4            key.Binding
	Sort5            key.Binding
	Sort6            key.Binding
}

var MainKeys = MainKeyMap{
	Quit:             key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Enter:            key.NewBinding(key.WithKeys("enter"), key.WithHelp("↵", "details")),
	Pull:             key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pull")),
	PullAll:          key.NewBinding(key.WithKeys("P"), key.WithHelp("P", "pull all")),
	Fetch:            key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "fetch")),
	FetchAll:         key.NewBinding(key.WithKeys("F"), key.WithHelp("F", "fetch all")),
	Search:           key.NewBinding(key.WithKeys("s", "/"), key.WithHelp("s", "search")),
	Escape:           key.NewBinding(key.WithKeys("esc")),
	Filter:           key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "filter")),
	SwitchDefault:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "switch default")),
	SwitchDefaultAll: key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "switch all default")),
	Sort1:            key.NewBinding(key.WithKeys("1"), key.WithHelp("1-6", "sort")),
	Sort2:            key.NewBinding(key.WithKeys("2")),
	Sort3:            key.NewBinding(key.WithKeys("3")),
	Sort4:            key.NewBinding(key.WithKeys("4")),
	Sort5:            key.NewBinding(key.WithKeys("5")),
	Sort6:            key.NewBinding(key.WithKeys("6")),
}

// DetailKeyMap defines key bindings for the detail screen.
type DetailKeyMap struct {
	Quit     key.Binding
	Back     key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
	Pull     key.Binding
	Push     key.Binding
	Tab1     key.Binding
	Tab2     key.Binding
	Tab3     key.Binding
	Tab4     key.Binding
}

var DetailKeys = DetailKeyMap{
	Quit:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	Back:     key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
	Tab:      key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	ShiftTab: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
	Pull:     key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pull")),
	Push:     key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "push")),
	Tab1:     key.NewBinding(key.WithKeys("1"), key.WithHelp("1-4", "tab")),
	Tab2:     key.NewBinding(key.WithKeys("2")),
	Tab3:     key.NewBinding(key.WithKeys("3")),
	Tab4:     key.NewBinding(key.WithKeys("4")),
}
