// Command git-as switches the git identity used for commits.
// Git runs it as a subcommand: `git as <profile>`.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

const usage = `usage: git as [<profile> [--global]]
       git as add <profile> --name <name> --email <email> [--signingkey <key>] [--sshcommand <cmd>]
       git as remove <profile>
       git as list

Switch the git identity used for commits.

  git as                      show the current identity and list profiles
  git as <profile>            apply a profile to the current repository
  git as <profile> --global   apply a profile globally
  git as add <profile> ...    create or replace a profile
  git as remove <profile>     delete a profile
  git as list                 list profiles

Profiles are stored in the global git config as profile.<name>.*.
`

type command struct {
	action     string // "show", "apply", "add", "remove", "list" or "help"
	profile    string
	global     bool
	userName   string
	email      string
	signingKey string
	sshCommand string
}

// usageError is a bad command line. It exits with status 2.
type usageError struct{ msg string }

func (e usageError) Error() string { return e.msg }

func usageErrorf(format string, a ...any) error {
	return usageError{fmt.Sprintf(format, a...)}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cmd, err := parseArgs(args)
	if err == nil {
		err = execute(cmd, stdout, stderr)
	}
	if err == nil {
		return 0
	}
	fmt.Fprintf(stderr, "git as: %v\n", err)
	var uerr usageError
	if errors.As(err, &uerr) {
		fmt.Fprintln(stderr, "run 'git as -h' for usage")
		return 2
	}
	return 1
}

func parseArgs(args []string) (command, error) {
	if slices.ContainsFunc(args, func(a string) bool { return a == "-h" || a == "--help" }) {
		return command{action: "help"}, nil
	}
	if len(args) == 0 {
		return command{action: "show"}, nil
	}

	switch args[0] {
	case "help":
		return command{action: "help"}, nil
	case "list":
		if len(args) > 1 {
			return command{}, usageErrorf("list takes no arguments")
		}
		return command{action: "list"}, nil
	case "remove":
		pos, _, err := parseFlags(args[1:], nil, nil)
		if err != nil {
			return command{}, err
		}
		name, err := oneProfileName(pos)
		return command{action: "remove", profile: name}, err
	case "add":
		pos, flags, err := parseFlags(args[1:], []string{"name", "email", "signingkey", "sshcommand"}, nil)
		if err != nil {
			return command{}, err
		}
		name, err := oneProfileName(pos)
		if err != nil {
			return command{}, err
		}
		cmd := command{
			action:     "add",
			profile:    name,
			userName:   flags["name"],
			email:      flags["email"],
			signingKey: flags["signingkey"],
			sshCommand: flags["sshcommand"],
		}
		if cmd.userName == "" {
			return command{}, usageErrorf("add: --name is required")
		}
		if cmd.email == "" {
			return command{}, usageErrorf("add: --email is required")
		}
		return cmd, nil
	default:
		pos, flags, err := parseFlags(args, nil, []string{"global"})
		if err != nil {
			return command{}, err
		}
		name, err := oneProfileName(pos)
		return command{action: "apply", profile: name, global: flags["global"] != ""}, err
	}
}

// parseFlags splits args into positional arguments and --flags. Flags in
// valueFlags take a value ("--name x" or "--name=x"); flags in boolFlags do not.
func parseFlags(args, valueFlags, boolFlags []string) ([]string, map[string]string, error) {
	var pos []string
	flags := map[string]string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			pos = append(pos, arg)
			continue
		}
		name, value, hasValue := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		switch {
		case !strings.HasPrefix(arg, "--"):
			return nil, nil, usageErrorf("unknown flag %s", arg)
		case slices.Contains(boolFlags, name):
			if hasValue {
				return nil, nil, usageErrorf("--%s takes no value", name)
			}
			flags[name] = "true"
		case slices.Contains(valueFlags, name):
			if !hasValue {
				if i+1 >= len(args) {
					return nil, nil, usageErrorf("--%s needs a value", name)
				}
				i++
				value = args[i]
			}
			flags[name] = value
		default:
			return nil, nil, usageErrorf("unknown flag %s", arg)
		}
	}
	return pos, flags, nil
}

// oneProfileName returns the single, valid profile name in pos.
func oneProfileName(pos []string) (string, error) {
	switch len(pos) {
	case 0:
		return "", usageErrorf("missing profile name")
	case 1:
		if err := validateProfileName(pos[0]); err != nil {
			return "", usageError{err.Error()}
		}
		return pos[0], nil
	default:
		return "", usageErrorf("too many arguments: %s", strings.Join(pos, " "))
	}
}

func execute(cmd command, stdout, stderr io.Writer) error {
	switch cmd.action {
	case "help":
		fmt.Fprint(stdout, usage)
		return nil
	case "show":
		return show(stdout)
	case "list":
		return list(stdout)
	case "apply":
		return apply(cmd, stdout, stderr)
	case "add":
		return add(cmd, stdout)
	case "remove":
		return remove(cmd, stdout)
	}
	return fmt.Errorf("unknown action %q", cmd.action)
}
