package app

import (
	"path/filepath"
	"sort"
	"strings"
)

func fileHasComments(path string, comments []Comment) bool {
	for _, c := range comments {
		if c.Path == path {
			return true
		}
	}
	return false
}

func buildTree(files []File, selectedPath string, viewed map[string]bool, comments []Comment) []TreeItem {
	root := &treeNode{Name: "", Path: "", IsDir: true, Children: map[string]*treeNode{}}
	for i := range files {
		pathSlash := files[i].PathSlash
		parts := strings.Split(pathSlash, "/")
		cur := root
		for j := 0; j < len(parts)-1; j++ {
			name := parts[j]
			if name == "" {
				continue
			}
			next, ok := cur.Children[name]
			if !ok {
				next = &treeNode{Name: name, Path: joinPath(cur.Path, name), IsDir: true, Children: map[string]*treeNode{}}
				cur.Children[name] = next
			}
			cur = next
		}
		fileName := parts[len(parts)-1]
		node := &treeNode{Name: fileName, Path: pathSlash, IsDir: false, File: &files[i]}
		cur.Children[fileName] = node
	}

	var items []TreeItem
	var walk func(n *treeNode, depth int)
	walk = func(n *treeNode, depth int) {
		if n != root {
			item := TreeItem{
				Name:     n.Name,
				Path:     "",
				Depth:    depth,
				IsDir:    n.IsDir,
				Selected: n.File != nil && n.File.Path == selectedPath,
			}
			if n.File != nil {
				item.Path = n.File.Path
				if viewed != nil {
					item.Viewed = viewed[n.File.Path]
				}
				item.HasComments = fileHasComments(n.File.Path, comments)
			}
			items = append(items, item)
		}
		children := make([]*treeNode, 0, len(n.Children))
		for _, child := range n.Children {
			children = append(children, child)
		}
		sort.Slice(children, func(i, j int) bool {
			if children[i].IsDir != children[j].IsDir {
				return children[i].IsDir
			}
			return children[i].Name < children[j].Name
		})
		for _, child := range children {
			walk(child, depth+1)
		}
	}
	walk(root, -1)
	return items
}

func buildGroupedTree(groups []Group, files []File, selectedPath string, viewed map[string]bool, comments []Comment) []TreeItem {
	var items []TreeItem
	grouped := make(map[string]bool)

	for _, g := range groups {
		// Determine if the selected path belongs to this group.
		groupActive := false
		for _, f := range g.Files {
			if f == selectedPath {
				groupActive = true
				break
			}
		}

		// Add group header.
		items = append(items, TreeItem{
			Name:        g.Name,
			Depth:       0,
			IsGroup:     true,
			GroupActive:  groupActive,
		})

		// Add files within this group.
		for _, gf := range g.Files {
			grouped[gf] = true
			file := findFileBySlash(files, gf)
			if file == nil {
				continue
			}
			item := TreeItem{
				Name:        filepath.Base(file.PathSlash),
				Path:        file.Path,
				Depth:       1,
				Selected:    file.Path == selectedPath,
				GroupName:   g.Name,
				HasComments: fileHasComments(file.Path, comments),
			}
			if viewed != nil {
				item.Viewed = viewed[file.Path]
			}
			items = append(items, item)
		}
	}

	// Collect ungrouped files.
	var ungrouped []File
	for _, f := range files {
		if !grouped[f.PathSlash] && !grouped[f.Path] {
			ungrouped = append(ungrouped, f)
		}
	}

	if len(ungrouped) > 0 {
		// Determine if the selected path belongs to the Other group.
		otherActive := false
		for _, f := range ungrouped {
			if f.Path == selectedPath {
				otherActive = true
				break
			}
		}

		items = append(items, TreeItem{
			Name:        "Other",
			Depth:       0,
			IsGroup:     true,
			GroupActive:  otherActive,
		})

		for _, f := range ungrouped {
			item := TreeItem{
				Name:        filepath.Base(f.PathSlash),
				Path:        f.Path,
				Depth:       1,
				Selected:    f.Path == selectedPath,
				GroupName:   "Other",
				HasComments: fileHasComments(f.Path, comments),
			}
			if viewed != nil {
				item.Viewed = viewed[f.Path]
			}
			items = append(items, item)
		}
	}

	return items
}

type treeNode struct {
	Name     string
	Path     string
	IsDir    bool
	Children map[string]*treeNode
	File     *File
}

func joinPath(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

func hasFile(files []File, path string) bool {
	for _, f := range files {
		if f.Path == path {
			return true
		}
	}
	return false
}

func findFile(files []File, path string) *File {
	for i := range files {
		if files[i].Path == path {
			return &files[i]
		}
	}
	return nil
}

func findFileBySlash(files []File, pathSlash string) *File {
	for i := range files {
		if files[i].PathSlash == pathSlash || files[i].Path == pathSlash {
			return &files[i]
		}
	}
	return nil
}
