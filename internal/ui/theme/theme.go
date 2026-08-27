// Package theme defines the gitpad color palette and shared styles.
// Dark-first (Catppuccin Mocha inspired) with light fallbacks.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	Text      = lipgloss.AdaptiveColor{Light: "#1e1e2e", Dark: "#cdd6f4"}
	Muted     = lipgloss.AdaptiveColor{Light: "#6c6f85", Dark: "#9399b2"}
	Dim       = lipgloss.AdaptiveColor{Light: "#9ca0b0", Dark: "#585b70"}
	Accent    = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"}
	OnAccent  = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#11111b"}
	Selection = lipgloss.AdaptiveColor{Light: "#d6e2ff", Dark: "#2c3e6e"}
	SelDim    = lipgloss.AdaptiveColor{Light: "#e6e9ef", Dark: "#2a2b3d"}
	Surface   = lipgloss.AdaptiveColor{Light: "#e6e9ef", Dark: "#1e1e2e"}
	Surface2  = lipgloss.AdaptiveColor{Light: "#dce0e8", Dark: "#313244"}
	Border    = lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#45475a"}

	Green   = lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}
	Red     = lipgloss.AdaptiveColor{Light: "#d20f39", Dark: "#f38ba8"}
	Yellow  = lipgloss.AdaptiveColor{Light: "#df8e1d", Dark: "#f9e2af"}
	Blue    = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"}
	Magenta = lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}
	Cyan    = lipgloss.AdaptiveColor{Light: "#179299", Dark: "#94e2d5"}
	Orange  = lipgloss.AdaptiveColor{Light: "#fe640b", Dark: "#fab387"}
	Pink    = lipgloss.AdaptiveColor{Light: "#ea76cb", Dark: "#f5c2e7"}
	Teal    = lipgloss.AdaptiveColor{Light: "#04a5e5", Dark: "#89dceb"}

	// Lanes are the commit-graph lane colors, cycled per lane.
	Lanes = []lipgloss.AdaptiveColor{Blue, Magenta, Cyan, Green, Yellow, Orange, Pink, Teal, Red}

	// Chip backgrounds (dark tints of the foreground color).
	chipHeadBg   = lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"}
	chipLocalBg  = lipgloss.AdaptiveColor{Light: "#dff3d9", Dark: "#2b3f30"}
	chipRemoteBg = lipgloss.AdaptiveColor{Light: "#ece0fb", Dark: "#3a2f4d"}
	chipTagBg    = lipgloss.AdaptiveColor{Light: "#fbefd2", Dark: "#4a4030"}
)

var (
	Base    = lipgloss.NewStyle().Foreground(Text)
	MutedSt = lipgloss.NewStyle().Foreground(Muted)
	DimSt   = lipgloss.NewStyle().Foreground(Dim)
	Bold    = lipgloss.NewStyle().Foreground(Text).Bold(true)
	AccentB = lipgloss.NewStyle().Foreground(Accent).Bold(true)

	Logo = lipgloss.NewStyle().Foreground(OnAccent).Background(Accent).Bold(true).Padding(0, 1)

	HeaderBar = lipgloss.NewStyle().Background(Surface)
	StatusBar = lipgloss.NewStyle().Background(Surface)
	FilterBar = lipgloss.NewStyle().Background(Surface)

	TabActive   = lipgloss.NewStyle().Foreground(Accent).Background(Surface2).Bold(true).Padding(0, 1)
	TabInactive = lipgloss.NewStyle().Foreground(Muted).Background(Surface).Padding(0, 1)

	FrameBorder       = lipgloss.NewStyle().Foreground(Border)
	FrameBorderActive = lipgloss.NewStyle().Foreground(Accent)
	FrameTitle        = lipgloss.NewStyle().Foreground(Muted).Bold(true)
	FrameTitleActive  = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	FrameCount        = lipgloss.NewStyle().Foreground(Dim)

	ChipHead   = lipgloss.NewStyle().Foreground(OnAccent).Background(chipHeadBg).Bold(true).Padding(0, 1)
	ChipLocal  = lipgloss.NewStyle().Foreground(Green).Background(chipLocalBg).Padding(0, 1)
	ChipRemote = lipgloss.NewStyle().Foreground(Magenta).Background(chipRemoteBg).Padding(0, 1)
	ChipTag    = lipgloss.NewStyle().Foreground(Yellow).Background(chipTagBg).Padding(0, 1)

	FilterChip       = lipgloss.NewStyle().Foreground(Muted).Background(Surface2).Padding(0, 1)
	FilterChipActive = lipgloss.NewStyle().Foreground(OnAccent).Background(Accent).Bold(true).Padding(0, 1)
	FilterChipFocus  = lipgloss.NewStyle().Foreground(Text).Background(Selection).Bold(true).Padding(0, 1)
	FilterInput      = lipgloss.NewStyle().Foreground(Text).Background(Surface2).Padding(0, 1)

	QuitKey = lipgloss.NewStyle().Foreground(Red).Bold(true)

	KeyHint    = lipgloss.NewStyle().Foreground(Accent).Bold(true)
	KeyLabel   = lipgloss.NewStyle().Foreground(Muted)
	ToastOK    = lipgloss.NewStyle().Foreground(OnAccent).Background(Green).Bold(true).Padding(0, 3)
	ToastErr   = lipgloss.NewStyle().Foreground(OnAccent).Background(Red).Bold(true).Padding(0, 3)
	ToastInfo  = lipgloss.NewStyle().Foreground(OnAccent).Background(Accent).Bold(true).Padding(0, 3)
	StateBadge = lipgloss.NewStyle().Foreground(OnAccent).Background(Orange).Bold(true).Padding(0, 1)

	ProgressBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Accent).Background(Surface).Padding(1, 3)

	MenuBox      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Accent).Background(Surface).Padding(0, 1)
	MenuTitle    = lipgloss.NewStyle().Foreground(Muted).Bold(true)
	MenuItem     = lipgloss.NewStyle().Foreground(Text)
	MenuItemSel  = lipgloss.NewStyle().Foreground(Text).Background(Selection).Bold(true)
	MenuDanger   = lipgloss.NewStyle().Foreground(Red)
	MenuDisabled = lipgloss.NewStyle().Foreground(Dim)
	MenuKey      = lipgloss.NewStyle().Foreground(Dim)
	MenuSep      = lipgloss.NewStyle().Foreground(Border)

	DialogBox       = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Accent).Background(Surface).Padding(1, 2)
	DialogBoxDanger = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Red).Background(Surface).Padding(1, 2)
	DialogTitle     = lipgloss.NewStyle().Foreground(Text).Bold(true)
	DialogBody      = lipgloss.NewStyle().Foreground(Muted)
	ButtonPrimary   = lipgloss.NewStyle().Foreground(OnAccent).Background(Accent).Bold(true).Padding(0, 2)
	ButtonDanger    = lipgloss.NewStyle().Foreground(OnAccent).Background(Red).Bold(true).Padding(0, 2)
	ButtonPlain     = lipgloss.NewStyle().Foreground(Muted).Background(Surface2).Padding(0, 2)

	DiffAdd    = lipgloss.NewStyle().Foreground(Green)
	DiffDel    = lipgloss.NewStyle().Foreground(Red)
	DiffHunk   = lipgloss.NewStyle().Foreground(Magenta).Bold(true)
	DiffMeta   = lipgloss.NewStyle().Foreground(Dim)
	DiffGutter = lipgloss.NewStyle().Foreground(Dim)
	DiffAddBg  = lipgloss.AdaptiveColor{Light: "#e3f5df", Dark: "#1f3324"}
	DiffDelBg  = lipgloss.AdaptiveColor{Light: "#fbe1e6", Dark: "#3b2129"}

	StatusA = lipgloss.NewStyle().Foreground(Green)
	StatusM = lipgloss.NewStyle().Foreground(Blue)
	StatusD = lipgloss.NewStyle().Foreground(Red)
	StatusR = lipgloss.NewStyle().Foreground(Magenta)
	StatusU = lipgloss.NewStyle().Foreground(Red).Bold(true)
	StatusQ = lipgloss.NewStyle().Foreground(Teal)
)

// Lane returns the style for graph lane color index i.
func Lane(i int) lipgloss.Style {
	if i < 0 {
		return Base
	}
	return lipgloss.NewStyle().Foreground(Lanes[i%len(Lanes)])
}

// Status returns the style for a name-status code.
func Status(code byte) lipgloss.Style {
	switch code {
	case 'A':
		return StatusA
	case 'D':
		return StatusD
	case 'R', 'C':
		return StatusR
	case 'U':
		return StatusU
	case '?':
		return StatusQ
	default:
		return StatusM
	}
}
