package render

import "math"

// SGR color codes. Only these two are used.
const (
	green = "32"
	red   = "31"
)

// UseColor decides whether to color the output for a --color mode of auto,
// always or never. auto colors only when stdout is a terminal and noColor
// (the value of $NO_COLOR) is empty.
func UseColor(mode string, tty bool, noColor string) bool {
	switch mode {
	case "always":
		return true
	case "auto":
		return tty && noColor == ""
	}
	return false
}

// paint wraps s in the SGR color code, or returns it unchanged if s is empty.
func paint(code, s string) string {
	if s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// GreenCells returns how many of a bar's n drawn cells are green:
// round(n × added / churn).
func GreenCells(n, added, churn int) int {
	if churn <= 0 {
		return 0
	}
	return int(math.Round(float64(n) * float64(added) / float64(churn)))
}

// colorBar colors bar's cells: the first GreenCells are green, the rest red.
func colorBar(bar string, added, churn int) string {
	cells := []rune(bar)
	g := GreenCells(len(cells), added, churn)
	return paint(green, string(cells[:g])) + paint(red, string(cells[g:]))
}
