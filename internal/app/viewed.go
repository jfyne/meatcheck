package app

// nextUnviewedFile returns the path of the next unviewed file starting from
// the file after SelectedPath, wrapping around. Returns "" if all files are
// viewed.
func nextUnviewedFile(model *ReviewModel) string {
	var paths []string

	if model.HasGroups {
		grouped := make(map[string]bool)
		for _, g := range model.Groups {
			paths = append(paths, g.Files...)
			for _, f := range g.Files {
				grouped[f] = true
			}
		}
		// Include ungrouped files ("Other" group) at the end.
		if model.Mode == ModeDiff {
			for _, df := range model.DiffFiles {
				if !grouped[df.Path] {
					paths = append(paths, df.Path)
				}
			}
		} else {
			for _, f := range model.Files {
				if !grouped[f.Path] && !grouped[f.PathSlash] {
					paths = append(paths, f.Path)
				}
			}
		}
	} else if model.Mode == ModeDiff {
		for _, df := range model.DiffFiles {
			paths = append(paths, df.Path)
		}
	} else {
		for _, f := range model.Files {
			paths = append(paths, f.Path)
		}
	}

	if len(paths) == 0 {
		return ""
	}

	// Find current index.
	currentIdx := -1
	for i, p := range paths {
		if p == model.SelectedPath {
			currentIdx = i
			break
		}
	}

	// Search starting from the next file, wrapping around.
	n := len(paths)
	for offset := 1; offset <= n; offset++ {
		idx := (currentIdx + offset) % n
		if currentIdx == -1 {
			// If selected path not found, start from 0.
			idx = offset - 1
			if idx >= n {
				break
			}
		}
		p := paths[idx]
		if model.Viewed == nil || !model.Viewed[p] {
			return p
		}
	}

	return ""
}
