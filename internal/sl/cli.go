package sl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const usageText = `Fetches upcoming SL journey alternatives.
When a route is saved, you can run ` + "`sl`" + ` without arguments.
Flags can be placed before or after route arguments.

Usage:
  sl <from> <to>
  sl -s <from> <to>
  sl <from> <to> -s
  sl
  sl -r
  sl -u | --upgrade
  sl -h | --help

Flags:
  -s  Save the provided route as your default (for plain ` + "`sl`" + `)
  -r  Reverse from/to (works with saved route or provided args)
  -u, --upgrade  Upgrades the CLI tool
  -h, --help  Show this help
`

func ParseCLIArgs(args []string) (CLIOptions, error) {
	var opts CLIOptions
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			opts.Positionals = append(opts.Positionals, args[i+1:]...)
			return opts, nil
		case a == "--help":
			opts.ShowHelp = true
		case a == "--upgrade":
			opts.Upgrade = true
		case strings.HasPrefix(a, "--"):
			return CLIOptions{}, fmt.Errorf("unknown flag: %s", a)
		case strings.HasPrefix(a, "-") && a != "-":
			for _, r := range a[1:] {
				switch r {
				case 's':
					opts.SavePair = true
				case 'r':
					opts.Reverse = true
				case 'h':
					opts.ShowHelp = true
				case 'u':
					opts.Upgrade = true
				default:
					return CLIOptions{}, fmt.Errorf("unknown flag: -%c", r)
				}
			}
		default:
			opts.Positionals = append(opts.Positionals, a)
		}
	}
	return opts, nil
}

func PrintUsage(out io.Writer) { _, _ = io.WriteString(out, usageText) }

func ResolveFromTo(args []string, reverse bool) (string, string, error) {
	var from, to string
	switch len(args) {
	case 0:
		cfg, err := loadConfig()
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return "", "", errors.New("no saved route found, use `sl -s <from> <to>`")
			}
			return "", "", fmt.Errorf("failed to load saved route: %w", err)
		}
		if cfg.From == "" || cfg.To == "" {
			return "", "", errors.New("saved route is empty, use `sl -s <from> <to>`")
		}
		from, to = cfg.From, cfg.To
	case 2:
		from, to = args[0], args[1]
	default:
		return "", "", errors.New("expected either no arguments or exactly two: <from> <to>")
	}
	if reverse {
		from, to = to, from
	}
	return from, to, nil
}
