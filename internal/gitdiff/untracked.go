package gitdiff

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Untracked lists untracked, non-ignored text files and counts all their lines
// as added. root is the repo root. Files that can't be read are skipped, with
// a warning for each.
func (g Git) Untracked(root string, pathspecs []string) (files []FileStat, warnings []string, err error) {
	if !slices.ContainsFunc(pathspecs, func(p string) bool { return !isExclude(p) }) {
		// Without a positive pathspec ls-files only lists the current
		// directory, while git diff --no-relative covers the whole repo.
		pathspecs = append([]string{":/"}, pathspecs...)
	}
	args := append([]string{"ls-files", "--others", "--exclude-standard", "--full-name", "-z", "--"}, pathspecs...)
	out, err := g.Run(args...)
	if err != nil {
		return nil, nil, err
	}
	for path := range strings.SplitSeq(string(out), "\x00") {
		if path == "" || strings.HasSuffix(path, "/") { // nested repository
			continue
		}
		n, err := countFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			var pe *os.PathError
			if errors.As(err, &pe) {
				err = pe.Err
			}
			warnings = append(warnings, fmt.Sprintf("cannot read %s: %v", path, err))
			continue
		}
		if n > 0 {
			files = append(files, FileStat{Path: path, Added: n})
		}
	}
	return files, warnings, nil
}

// countFile returns the number of lines in the file at path, or 0 if it is
// not a regular file, is binary or is empty.
func countFile(path string) (int, error) {
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() {
		return 0, err
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return countLines(f)
}

// countLines streams r and returns the number of '\n', plus 1 if r is
// non-empty and doesn't end in '\n'. It returns 0 if the first 8 KiB contain
// a NUL byte (binary).
func countLines(r io.Reader) (int, error) {
	br := bufio.NewReaderSize(r, 8<<10)
	head, err := br.Peek(8 << 10)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if bytes.IndexByte(head, 0) >= 0 {
		return 0, nil
	}
	lines, last := 0, byte('\n')
	buf := make([]byte, 32<<10)
	for {
		n, err := br.Read(buf)
		if n > 0 {
			lines += bytes.Count(buf[:n], []byte{'\n'})
			last = buf[n-1]
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
	}
	if last != '\n' {
		lines++
	}
	return lines, nil
}
