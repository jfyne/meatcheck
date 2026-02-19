package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jfyne/meatcheck/internal/app"
)

type listFlag []string

func (l *listFlag) String() string {
	return strings.Join(*l, ",")
}

func (l *listFlag) Set(value string) error {
	*l = append(*l, value)
	return nil
}

func main() {
	var (
		host      = flag.String("host", "127.0.0.1", "host to bind")
		port      = flag.Int("port", 0, "port to bind (0 = random)")
		prompt    = flag.String("prompt", "", "review prompt/question to display at top")
		diff      = flag.String("diff", "", "path to unified diff file (or pipe via stdin)")
		ranges    listFlag
		showHelp  = flag.Bool("help", false, "show help")
		showSkill = flag.Bool("skill", false, "print agent skill markdown")
	)
	flag.Var(&ranges, "range", "file section to render (path:start-end), repeatable")
	flag.Parse()

	if *showHelp {
		app.PrintHelp(os.Stdout)
		return
	}
	if *showSkill {
		app.PrintSkill(os.Stdout)
		return
	}

	stdDiff, err := app.ReadStdDiff()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if flag.NArg() == 0 && stdDiff == "" && *diff == "" {
		fmt.Fprintln(os.Stderr, "usage: meatcheck <file1> <file2> ...")
		fmt.Fprintln(os.Stderr, "run with --help for more information")
		os.Exit(2)
	}

	rangesMap, err := app.ParseRangeFlag(ranges)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg := app.Config{
		Host:    *host,
		Port:    *port,
		Paths:   flag.Args(),
		Prompt:  *prompt,
		Diff:    *diff,
		Ranges:  rangesMap,
		StdDiff: stdDiff,
	}
	if err := app.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
