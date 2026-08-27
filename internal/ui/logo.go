package ui

import (
	_ "embed"
	"encoding/base64"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jinhyo-dev/gitpad/internal/ui/theme"
)

// The header shows the gitpad logo as an image in terminals that can draw
// them (iTerm2, WezTerm, Kitty, Ghostty, Warp…) and, everywhere else, drawn
// with the same dot/curve glyphs as the commit graph across the two header rows.

//go:embed logo.png
var logoPNG []byte

type logoMode int

const (
	logoText logoMode = iota
	logoITerm
	logoKitty
	logoOff
)

// logoCells is how many terminal columns the image occupies. Both protocols
// advance the cursor past the image, but the escape sequence itself has zero
// printable width, so the header layout compensates with logoHidden().
const logoCells = 2

func detectLogoMode() logoMode {
	switch strings.ToLower(os.Getenv("GITPAD_LOGO")) {
	case "off", "none":
		return logoOff
	case "text":
		return logoText
	case "iterm", "iterm2":
		return logoITerm
	case "kitty":
		return logoKitty
	}
	if fi, err := os.Stdout.Stat(); err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return logoText // not a terminal (tests, --snapshot)
	}
	if os.Getenv("TMUX") != "" || os.Getenv("STY") != "" {
		return logoText // multiplexers need passthrough wrappers; keep it simple
	}
	if os.Getenv("KITTY_WINDOW_ID") != "" || strings.Contains(os.Getenv("TERM"), "kitty") {
		return logoKitty
	}
	switch strings.ToLower(os.Getenv("TERM_PROGRAM")) {
	case "iterm.app":
		return logoITerm
	case "wezterm":
		return logoITerm
	case "ghostty":
		return logoKitty
	case "warpterminal":
		return logoKitty
	}
	return logoText
}

var logoModeCached = detectLogoMode()

// logoMark renders the logo for the header.
func logoMark() string {
	b64 := base64.StdEncoding.EncodeToString(logoPNG)
	switch logoModeCached {
	case logoOff:
		return ""
	case logoITerm:
		return "\x1b]1337;File=inline=1;width=" + itoa(logoCells) + ";height=1;preserveAspectRatio=1:" + b64 + "\a"
	case logoKitty:
		// Transmit + display a PNG (f=100) sized to logoCells×1 cells with a
		// fixed image id so repaints replace rather than accumulate; q=2
		// suppresses responses that would otherwise land in the input stream.
		return "\x1b_Gf=100,a=T,i=31337,q=2,c=" + itoa(logoCells) + ",r=1;" + b64 + "\x1b\\"
	default:
		return logoArtRow(0)
	}
}

// Text fallback, drawn only with glyphs the commit graph already relies on
// (quadrant blocks render inconsistently across fonts):
//
//	●╮
//	●╰●
func logoArtRow(row int) string {
	blue := lipgloss.NewStyle().Foreground(theme.Blue)
	purple := lipgloss.NewStyle().Foreground(theme.Magenta)
	if row == 0 {
		return blue.Render("●╮") + " "
	}
	return blue.Render("●╰") + purple.Render("●")
}

// logoFilterPrefix is what the filter row shows under the header logo: the
// second row of the text art, or nothing when an image logo is used.
func logoFilterPrefix() string {
	if logoModeCached == logoText {
		return logoArtRow(1) + " "
	}
	return ""
}

// logoHidden is the number of columns the logo takes on screen beyond what
// ansi width calculation sees (image protocols only).
func logoHidden() int {
	if logoModeCached == logoITerm || logoModeCached == logoKitty {
		return logoCells
	}
	return 0
}
