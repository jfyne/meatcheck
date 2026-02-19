package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpkeskin/gotoon"
)

func loadFiles(paths []string) ([]File, error) {
	files := make([]File, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
		files = append(files, File{
			Path:      path,
			PathSlash: filepath.ToSlash(path),
			Lines:     lines,
		})
	}
	return files, nil
}

func ReadStdDiff() (string, error) {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", nil
	}
	reader := bufio.NewReader(os.Stdin)
	var b strings.Builder
	for {
		chunk, err := reader.ReadString('\n')
		b.WriteString(chunk)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
	}
	return b.String(), nil
}

func ParseRangeFlag(values []string) (map[string][]LineRange, error) {
	if len(values) == 0 {
		return nil, nil
	}
	ranges := make(map[string][]LineRange)
	for _, val := range values {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		parts := strings.SplitN(val, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid range: %s", val)
		}
		path := parts[0]
		r := parts[1]
		seg := strings.SplitN(r, "-", 2)
		if len(seg) != 2 {
			return nil, fmt.Errorf("invalid range: %s", val)
		}
		start := mustAtoi(seg[0])
		end := mustAtoi(seg[1])
		if start == 0 || end == 0 {
			return nil, fmt.Errorf("invalid range: %s", val)
		}
		if end < start {
			start, end = end, start
		}
		ranges[path] = append(ranges[path], LineRange{Start: start, End: end})
	}
	return ranges, nil
}

func emitToon(w io.Writer, comments []Comment) error {
	doc := map[string]any{
		"comments": comments,
	}
	encoded, err := gotoon.Encode(doc)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, encoded)
	return err
}
