// Package tree builds a directory tree of line counts from per-file stats.
// It contains only pure functions.
package tree

import "strings"

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
