// Package tree builds a directory tree of line counts from per-file stats.
// It contains only pure functions.
package tree

import (
	"cmp"
	"slices"
	"strings"
)

// FileStat is the line count for one changed file.
type FileStat struct {
	Path    string // slash-separated, relative to repo root
	Added   int
	Deleted int
}

// Node is a directory or file in the tree, with totals for everything below it.
type Node struct {
	Name     string // display label; may hold a collapsed chain like "internal/util"
	IsDir    bool
	Added    int // sum over everything under this node
	Deleted  int
	Files    int // count of changed files under this node
	Children []*Node
}

// Churn is the number of added plus deleted lines.
func (n *Node) Churn() int { return n.Added + n.Deleted }

// Build returns the repo root node for files. The root is never printed as a
// row; its totals go in the header. Children keep the order of first
// appearance in files.
func Build(files []FileStat) *Node {
	root := &Node{IsDir: true}
	dirs := map[string]*Node{"": root}
	for _, f := range files {
		n := root
		n.add(f)
		names := strings.Split(f.Path, "/")
		for i, name := range names[:len(names)-1] {
			key := strings.Join(names[:i+1], "/")
			d := dirs[key]
			if d == nil {
				d = &Node{Name: name, IsDir: true}
				dirs[key] = d
				n.Children = append(n.Children, d)
			}
			d.add(f)
			n = d
		}
		leaf := &Node{Name: names[len(names)-1]}
		leaf.add(f)
		n.Children = append(n.Children, leaf)
	}
	return root
}

func (n *Node) add(f FileStat) {
	n.Added += f.Added
	n.Deleted += f.Deleted
	n.Files++
}

// Options controls how Shape turns the full tree into the rows to print.
type Options struct {
	Files  bool // show file leaves below the root
	ByName bool // sort by name instead of churn
	Depth  int  // levels to keep; 0 = unlimited
}

// Shape applies the steps of spec §6.1 to root, in order, modifying it in
// place: hide files, collapse chains, sort, limit depth.
func Shape(root *Node, o Options) {
	if !o.Files {
		HideFiles(root)
	}
	Collapse(root)
	Sort(root, o.ByName)
	LimitDepth(root, o.Depth)
}

// HideFiles removes file nodes everywhere below root's direct children, so
// files at the repo root stay. Totals are unchanged.
func HideFiles(root *Node) {
	for _, c := range root.Children {
		hideFiles(c)
	}
}

func hideFiles(n *Node) {
	n.Children = slices.DeleteFunc(n.Children, func(c *Node) bool { return !c.IsDir })
	for _, c := range n.Children {
		hideFiles(c)
	}
}

// Collapse merges every directory below root whose only child is a
// directory into a single node labeled like "internal/util". The merged node
// keeps the outer directory's totals, so a hidden file in the outer directory
// still counts toward the row. root itself is never merged.
func Collapse(root *Node) {
	for _, c := range root.Children {
		for c.IsDir && len(c.Children) == 1 && c.Children[0].IsDir {
			c.Name += "/" + c.Children[0].Name
			c.Children = c.Children[0].Children
		}
		Collapse(c)
	}
}

// Sort orders siblings at every level by churn (highest first, ties by name)
// or, with byName, by name. Directories and files are mixed.
func Sort(n *Node, byName bool) {
	slices.SortStableFunc(n.Children, func(a, b *Node) int {
		if !byName {
			if c := cmp.Compare(b.Churn(), a.Churn()); c != 0 {
				return c
			}
		}
		return strings.Compare(a.Name, b.Name)
	})
	for _, c := range n.Children {
		Sort(c, byName)
	}
}

// LimitDepth drops nodes below level depth, counting root's children as
// level 1. depth 0 keeps everything. Totals are unchanged.
func LimitDepth(root *Node, depth int) {
	if depth <= 0 {
		return
	}
	for _, c := range root.Children {
		if depth == 1 {
			c.Children = nil
		} else {
			LimitDepth(c, depth-1)
		}
	}
}
