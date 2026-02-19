package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jfyne/meatcheck/internal/app"
)

func main() {
	var (
		host      = flag.String("host", "127.0.0.1", "host to bind")
		port      = flag.Int("port", 0, "port to bind (0 = random)")
		prompt    = flag.String("prompt", "", "review prompt/question to display at top")
		showHelp  = flag.Bool("help", false, "show help")
		showSkill = flag.Bool("skill", false, "print agent skill markdown")
	)
	flag.Parse()

	if *showHelp {
		app.PrintHelp(os.Stdout)
		return
	}
	if *showSkill {
		app.PrintSkill(os.Stdout)
		return
	}

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: meatcheck <file1> <file2> ...")
		fmt.Fprintln(os.Stderr, "run with --help for more information")
		os.Exit(2)
	}

	cfg := app.Config{
		Host:   *host,
		Port:   *port,
		Paths:  flag.Args(),
		Prompt: *prompt,
	}
	if err := app.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
