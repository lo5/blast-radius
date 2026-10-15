package render

import (
	"math"
	"strings"
)

// Bar widths are limited to this range of cells.
const (
	minBarWidth = 10
	maxBarWidth = 40
)

// partials are the partial cells for 1 to 7 eighths.
var partials = []string{"▏", "▎", "▍", "▌", "▋", "▊", "▉"}

// Eighths returns the bar length in eighths of a cell for churn on a scale
// where maxChurn fills w cells: round(churn / maxChurn × w × 8), at least 1
// when churn > 0.
func Eighths(churn, maxChurn, w int) int {
	if churn <= 0 || maxChurn <= 0 {
		return 0
	}
	return max(1, int(math.Round(float64(churn)/float64(maxChurn)*float64(w)*8)))
}

// Bar draws a bar of the given length in eighths: full cells as █, then one
// partial cell if needed. It is not padded.
func Bar(eighths int) string {
	s := strings.Repeat("█", eighths/8)
	if r := eighths % 8; r > 0 {
		s += partials[r-1]
	}
	return s
}
