// Package render formats a tree of line counts as terminal text.
// It contains only pure functions.
package render

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/lo5/blast-radius/internal/tree"
)

// Render returns the full output: the header, a blank line, then one row per
// node below root, or "no changes" if there is nothing to show. Rows are
// fitted to a terminal of the given width. With color, bars and counts are
// green for added lines and red for deleted lines.
func Render(label string, root *tree.Node, termWidth int, color bool) string {
	var b strings.Builder
	b.WriteString(Header(label, root))
	b.WriteString("\n\n")
	if root.Files == 0 {
		b.WriteString("no changes\n")
		return b.String()
	}
	rows := flatten(nil, root.Children, "", true)
	var nameW, addW, delW, filesW, pctW, maxChurn int
	for i := range rows {
		r := &rows[i]
		r.add = "+" + Thousands(r.node.Added)
		r.del = "-" + Thousands(r.node.Deleted)
		r.files = Plural(r.node.Files, "file")
		r.pct = Percent(r.node.Churn(), root.Churn())
		nameW = max(nameW, width(r.prefix+r.label))
		addW = max(addW, width(r.add))
		delW = max(delW, width(r.del))
		filesW = max(filesW, width(r.files))
		pctW = max(pctW, width(r.pct))
		maxChurn = max(maxChurn, r.node.Churn())
	}
	// Everything but the name and bar columns, plus the four separators.
	rest := addW + 1 + delW + filesW + pctW + 4*2
	if nameW+rest+minBarWidth > termWidth {
		nameW = min(nameW, termWidth*2/5)
	}
	barW := min(max(termWidth-nameW-rest, minBarWidth), maxBarWidth)
	for _, r := range rows {
		bar, add, del := Bar(Eighths(r.node.Churn(), maxChurn, barW)), r.add, r.del
		// Pad by the plain text so SGR codes don't count toward widths.
		barPad, addPad, delPad := pad(bar, barW), pad(add, addW), pad(del, delW)
		if color {
			bar, add, del = colorBar(bar, r.node.Added, r.node.Churn()), paint(green, add), paint(red, del)
		}
		b.WriteString(padRight(truncate(r.prefix, r.label, nameW), nameW))
		b.WriteString("  " + bar + barPad)
		b.WriteString("  " + addPad + add + " " + delPad + del)
		b.WriteString("  " + padLeft(r.files, filesW))
		b.WriteString("  " + padLeft(r.pct, pctW))
		b.WriteString("\n")
	}
	return b.String()
}

// Header returns `<label> · <F> files · +<added> -<deleted>`.
func Header(label string, root *tree.Node) string {
	return fmt.Sprintf("%s · %s · +%s -%s",
		label, Plural(root.Files, "file"), Thousands(root.Added), Thousands(root.Deleted))
}

type row struct {
	prefix, label        string // tree prefix; name, with / for directories
	node                 *tree.Node
	add, del, files, pct string
}

// flatten appends a row for each node and its descendants, depth first.
// Top-level rows have no tree prefix.
func flatten(rows []row, nodes []*tree.Node, indent string, top bool) []row {
	for i, n := range nodes {
		branch, cont := "├─ ", "│  "
		if i == len(nodes)-1 {
			branch, cont = "└─ ", "   "
		}
		if top {
			branch, cont = "", ""
		}
		label := n.Name
		if n.IsDir {
			label += "/"
		}
		rows = append(rows, row{prefix: indent + branch, label: label, node: n})
		rows = flatten(rows, n.Children, indent+cont, false)
	}
	return rows
}

// Thousands formats n with comma thousands separators, e.g. 1,204.
func Thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

// Plural returns "1 <word>" or "<n> <word>s".
func Plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return Thousands(n) + " " + word + "s"
}

// Percent returns round(churn / total × 100)%, or "<1%" for a non-zero
// share that rounds below 1.
func Percent(churn, total int) string {
	if total == 0 {
		return "0%"
	}
	p := int(math.Round(float64(churn) * 100 / float64(total)))
	if p < 1 && churn > 0 {
		return "<1%"
	}
	return strconv.Itoa(p) + "%"
}

// truncate returns prefix+label, shortening label from the end with … so the
// result is at most w wide. The prefix is always kept.
func truncate(prefix, label string, w int) string {
	if width(prefix+label) <= w {
		return prefix + label
	}
	keep := max(0, w-width(prefix)-1)
	return prefix + string([]rune(label)[:keep]) + "…"
}

func width(s string) int { return utf8.RuneCountInString(s) }

// pad returns the spaces needed to pad s to w cells.
func pad(s string, w int) string { return strings.Repeat(" ", max(0, w-width(s))) }

func padRight(s string, w int) string { return s + pad(s, w) }

func padLeft(s string, w int) string { return pad(s, w) + s }
