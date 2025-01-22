package chat

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

func formatTime(time time.Time) string {
	return fmt.Sprintf("%02d:%02d UTC", time.UTC().Hour(), time.UTC().Minute())
}

func clamp(value, min, max int) int {
	if max < min {
		return min
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func createAreaInput(placeholder string, limit int, initialWidth int) textarea.Model {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.CharLimit = limit
	ta.ShowLineNumbers = false

	ta.MaxHeight = 2
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()

	ta.Prompt = ""

	ta.SetHeight(2)
	ta.SetWidth(initialWidth)

	return ta
}
