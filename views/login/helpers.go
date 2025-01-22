package login

import "github.com/charmbracelet/bubbles/textinput"

func createTextInput(placeholder string, limit int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = limit
	ti.Prompt = ""
	ti.Width = 10
	return ti
}
