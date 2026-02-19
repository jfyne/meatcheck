package app

import (
	"sort"
	"strings"
)

func buildTree(files []File, selectedPath string) []TreeItem {
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
