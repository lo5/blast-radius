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
// node below root, or "no changes" if there is nothing to show.
func Render(label string, root *tree.Node) string {
	var b strings.Builder
	b.WriteString(Header(label, root))
	b.WriteString("\n\n")
	if root.Files == 0 {
		b.WriteString("no changes\n")
		return b.String()
	}
	rows := flatten(nil, root.Children, "", true)
	var nameW, addW, delW, filesW, pctW int
	for i := range rows {
		r := &rows[i]
		r.add = "+" + Thousands(r.node.Added)
		r.del = "-" + Thousands(r.node.Deleted)
		r.files = Plural(r.node.Files, "file")
		r.pct = Percent(r.node.Churn(), root.Churn())
		nameW = max(nameW, width(r.name))
		addW = max(addW, width(r.add))
		delW = max(delW, width(r.del))
		filesW = max(filesW, width(r.files))
		pctW = max(pctW, width(r.pct))
	}
	for _, r := range rows {
		b.WriteString(padRight(r.name, nameW))
		b.WriteString("  " + padLeft(r.add, addW) + " " + padLeft(r.del, delW))
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
	name                 string // tree prefix plus label
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
		rows = append(rows, row{name: indent + branch + label, node: n})
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

func width(s string) int { return utf8.RuneCountInString(s) }

func padRight(s string, w int) string { return s + strings.Repeat(" ", max(0, w-width(s))) }

func padLeft(s string, w int) string { return strings.Repeat(" ", max(0, w-width(s))) + s }
