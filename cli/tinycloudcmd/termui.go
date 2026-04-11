package tinycloudcmd

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const tinyCloudBanner = `   __  _                  __                __
  / /_(_)___  __  _______/ /___  __  ______/ /
 / __/ / __ \/ / / / ___/ / __ \/ / / / __  / 
/ /_/ / / / / /_/ / /__/ / /_/ / /_/ / /_/ /  
\__/_/_/ /_/\__, /\___/_/\____/\__,_/\__,_/   
           /____/`

const (
	ansiReset  = "\x1b[0m"
	ansiGreen  = "\x1b[32m"
	ansiRed    = "\x1b[31m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiDim    = "\x1b[2m"
)

type terminalUI struct {
	w           io.Writer
	interactive bool
	color       bool
}

type table struct {
	headers []string
	rows    [][]string
}

func newTerminalUI(w io.Writer) terminalUI {
	interactive := isInteractiveWriter(w)
	color := interactive && os.Getenv("NO_COLOR") == ""
	return terminalUI{
		w:           w,
		interactive: interactive,
		color:       color,
	}
}

func isInteractiveWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	term := strings.TrimSpace(strings.ToLower(os.Getenv("TERM")))
	if term == "dumb" {
		return false
	}
	return true
}

func (ui terminalUI) renderBanner() string {
	if !ui.interactive {
		return ""
	}
	return tinyCloudBanner + "\n\n"
}

func (ui terminalUI) success(value string) string {
	return ui.iconize("✓", value, ansiGreen)
}

func (ui terminalUI) failure(value string) string {
	return ui.iconize("✗", value, ansiRed)
}

func (ui terminalUI) warning(value string) string {
	return ui.iconize("‼", value, ansiYellow)
}

func (ui terminalUI) progress(value string) string {
	return ui.iconize("…", value, ansiDim)
}

func (ui terminalUI) active(value string) string {
	return ui.iconize("●", value, ansiCyan)
}

func (ui terminalUI) inactive(value string) string {
	return ui.iconize("○", value, ansiDim)
}

func (ui terminalUI) iconize(icon, value, color string) string {
	if !ui.color {
		return icon + " " + value
	}
	return color + icon + ansiReset + " " + value
}

func (ui terminalUI) colorize(value, color string) string {
	if !ui.color {
		return value
	}
	return color + value + ansiReset
}

func (ui terminalUI) section(title string) string {
	return title + "\n"
}

func (ui terminalUI) keyValues(items [][2]string) string {
	if len(items) == 0 {
		return ""
	}
	width := 0
	for _, item := range items {
		if len(item[0]) > width {
			width = len(item[0])
		}
	}

	var builder strings.Builder
	for _, item := range items {
		builder.WriteString("  ")
		builder.WriteString(item[0])
		builder.WriteString(strings.Repeat(" ", width-len(item[0])+2))
		builder.WriteString(item[1])
		builder.WriteString("\n")
	}
	return builder.String()
}

func (ui terminalUI) renderTable(t table) string {
	if len(t.headers) == 0 {
		return ""
	}
	widths := make([]int, len(t.headers))
	for i, header := range t.headers {
		widths[i] = len(stripANSI(header))
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if cellWidth := len(stripANSI(cell)); cellWidth > widths[i] {
				widths[i] = cellWidth
			}
		}
	}

	var builder strings.Builder
	for i, header := range t.headers {
		if i > 0 {
			builder.WriteString("  ")
		}
		builder.WriteString(padRight(header, widths[i]))
	}
	builder.WriteString("\n")
	for i, width := range widths {
		if i > 0 {
			builder.WriteString("  ")
		}
		builder.WriteString(strings.Repeat("-", width))
	}
	builder.WriteString("\n")
	for _, row := range t.rows {
		for i := range t.headers {
			if i > 0 {
				builder.WriteString("  ")
			}
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			builder.WriteString(padRight(cell, widths[i]))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func padRight(value string, width int) string {
	padding := width - len(stripANSI(value))
	if padding <= 0 {
		return value
	}
	return value + strings.Repeat(" ", padding)
}

func stripANSI(value string) string {
	var builder strings.Builder
	inEscape := false
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if !inEscape && ch == 0x1b {
			inEscape = true
			continue
		}
		if inEscape {
			if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
				inEscape = false
			}
			continue
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func writeString(w io.Writer, value string) error {
	_, err := io.WriteString(w, value)
	return err
}

func joinLines(lines ...string) string {
	values := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		values = append(values, strings.TrimRight(line, "\n"))
	}
	if len(values) == 0 {
		return ""
	}
	return strings.Join(values, "\n") + "\n"
}

func formatOptionalValue(prefix, value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return fmt.Sprintf("%s%s", prefix, value)
}
