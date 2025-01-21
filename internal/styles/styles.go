package styles

import "github.com/charmbracelet/lipgloss"

type ClientStyles struct {
	RegularTxt     lipgloss.Style
	BoldRegularTxt lipgloss.Style
	PlaceholderTxt lipgloss.Style
	Button         lipgloss.Style
	ActiveButton   lipgloss.Style
	DialogBox      lipgloss.Style

	PrimaryColor lipgloss.Color
	GreyColor    lipgloss.Color
	MutedColor   lipgloss.Color
}

func NewClientStyles(renderer *lipgloss.Renderer) *ClientStyles {
	txtStyle := renderer.NewStyle().Foreground(lipgloss.Color("15"))
	boldTxtStyle := renderer.NewStyle().Inherit(txtStyle).Bold(true)
	placeholderTxtStyle := renderer.NewStyle().Foreground(lipgloss.Color("240"))

	primaryColor := lipgloss.Color("#F25D94")
	greyColor := lipgloss.Color("#888B7E")
	mutedColor := lipgloss.Color("240")

	buttonStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFF7DB")).
		Background(greyColor).
		Padding(0, 2)
	activeButtonStyle := buttonStyle.
		Foreground(lipgloss.Color("#FFF7DB")).
		Background(lipgloss.Color(primaryColor))

	dialogBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFF7DB")).
		Padding(1, 0).
		BorderTop(true).
		BorderLeft(true).
		BorderRight(true).
		BorderBottom(true)

	return &ClientStyles{
		RegularTxt:     txtStyle,
		BoldRegularTxt: boldTxtStyle,
		PlaceholderTxt: placeholderTxtStyle,
		Button:         buttonStyle,
		ActiveButton:   activeButtonStyle,
		DialogBox:      dialogBoxStyle,

		PrimaryColor: primaryColor,
		GreyColor:    greyColor,
		MutedColor:   mutedColor,
	}
}
