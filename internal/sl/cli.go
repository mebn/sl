package sl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func parseCLIArgs(args []string) (cliOptions, error) {
	var opts cliOptions

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			opts.Positionals = append(opts.Positionals, args[i+1:]...)
			break
		}
		if arg == "--help" {
			opts.ShowHelp = true
			continue
		}

		if strings.HasPrefix(arg, "-") && arg != "-" {
			if strings.HasPrefix(arg, "--") {
				return cliOptions{}, fmt.Errorf("unknown flag: %s", arg)
			}
			for _, flagChar := range arg[1:] {
				switch flagChar {
				case 's':
					opts.SavePair = true
				case 'r':
					opts.Reverse = true
				case 'h':
					opts.ShowHelp = true
				default:
					return cliOptions{}, fmt.Errorf("unknown flag: -%c", flagChar)
				}
			}
			continue
		}

		opts.Positionals = append(opts.Positionals, arg)
	}

	return opts, nil
}

func printUsage(out io.Writer) {
	fmt.Fprintf(out, "SL - public transit trips between two places\n\n")
	fmt.Fprintf(out, "Fetches upcoming SL journey alternatives and prints legs with times.\n")
	fmt.Fprintf(out, "It filters out tiny metro-hop detours when a direct walk alternative exists.\n")
	fmt.Fprintf(out, "When a route is saved, you can run `sl` without arguments.\n\n")
	fmt.Fprintf(out, "Usage:\n")
	fmt.Fprintf(out, "  sl <from> <to>\n")
	fmt.Fprintf(out, "  sl -s <from> <to>\n")
	fmt.Fprintf(out, "  sl <from> <to> -s\n")
	fmt.Fprintf(out, "  sl\n")
	fmt.Fprintf(out, "  sl -r\n")
	fmt.Fprintf(out, "  sl -h | --help\n\n")
	fmt.Fprintf(out, "Flags:\n")
	fmt.Fprintf(out, "  -s  Save the provided route as your default (for plain `sl`)\n")
	fmt.Fprintf(out, "  -r  Reverse from/to (works with saved route or provided args)\n")
	fmt.Fprintf(out, "  -h, --help  Show this help\n\n")
	fmt.Fprintf(out, "Examples:\n")
	fmt.Fprintf(out, "  sl Odenplan Slussen\n")
	fmt.Fprintf(out, "  sl -s \"Langsjo torg\" \"Sveavagen 20, Stockholm\"\n")
	fmt.Fprintf(out, "  sl -r\n\n")
	fmt.Fprintf(out, "Note: flags can be placed before or after route arguments.\n")
}

func resolveFromTo(args []string, reverse bool) (string, string, error) {
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
