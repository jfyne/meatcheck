package app

import (
	_ "embed"
	"fmt"
	"io"
)

//go:embed skill.md
var skillMarkdown string

func PrintSkill(w io.Writer) {
	fmt.Fprint(w, skillMarkdown)
}
