package main

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// show prints the current identity, the profile it matches, and all profiles.
func show(w io.Writer) error {
	name, email, err := currentIdentity()
	if err != nil {
		return err
	}
	profiles, err := loadProfiles()
	if err != nil {
		return err
	}

	if name == "" && email == "" {
		fmt.Fprintln(w, "current: (no identity set)")
	} else {
		fmt.Fprintf(w, "current: %s", formatIdentity(name, email))
		var matches []string
		for _, p := range profiles {
			if p.userName == name && p.email == email {
				matches = append(matches, p.name)
			}
		}
		if len(matches) > 0 {
			fmt.Fprintf(w, " (profile: %s)", strings.Join(matches, ", "))
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w)
	if len(profiles) > 0 {
		fmt.Fprintln(w, "profiles:")
	}
	printProfiles(w, profiles, name, email)
	return nil
}

// list prints all profiles. A "*" marks profiles that match the current identity.
func list(w io.Writer) error {
	name, email, err := currentIdentity()
	if err != nil {
		return err
	}
	profiles, err := loadProfiles()
	if err != nil {
		return err
	}
	printProfiles(w, profiles, name, email)
	return nil
}

func printProfiles(w io.Writer, profiles []profile, name, email string) {
	if len(profiles) == 0 {
		fmt.Fprintln(w, "no profiles; create one with: git as add <profile> --name <name> --email <email>")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, p := range profiles {
		mark := " "
		if p.userName == name && p.email == email {
			mark = "*"
		}
		fmt.Fprintf(tw, "%s %s\t%s\n", mark, p.name, formatIdentity(p.userName, p.email))
	}
	tw.Flush()
}

// apply writes a profile's identity to the local or global config.
func apply(cmd command, stdout, stderr io.Writer) error {
	p, err := findProfile(cmd.profile)
	if err != nil {
		return err
	}
	if p.userName == "" || p.email == "" {
		return fmt.Errorf("profile %q has no name or email; fix it with 'git as add'", p.name)
	}
	scope := "global"
	if !cmd.global {
		if !insideRepo() {
			return errors.New("not inside a git repository (use --global to apply the profile globally)")
		}
		scope = "local"
	}

	// Optional values that the profile does not define are unset, so a
	// previous profile's key or SSH command does not stay active.
	settings := []struct{ key, value string }{
		{"user.name", p.userName},
		{"user.email", p.email},
		{"user.signingkey", p.signingKey},
		{"core.sshCommand", p.sshCommand},
	}
	for _, s := range settings {
		if err := configSetOrUnset(scope, s.key, s.value); err != nil {
			return err
		}
	}
	fmt.Fprintf(stdout, "now using %s: %s (%s)\n", p.name, formatIdentity(p.userName, p.email), scope)

	if cmd.global && insideRepo() {
		localName, _ := configGet("local", "user.name")
		localEmail, _ := configGet("local", "user.email")
		if localName != "" || localEmail != "" {
			fmt.Fprintln(stderr, "note: this repository sets its own identity, which overrides the global one here")
		}
	}
	return nil
}

// add creates a profile, or replaces every field of an existing one.
func add(cmd command, stdout io.Writer) error {
	fields := []struct{ field, value string }{
		{"name", cmd.userName},
		{"email", cmd.email},
		{"signingkey", cmd.signingKey},
		{"sshcommand", cmd.sshCommand},
	}
	for _, f := range fields {
		if err := configSetOrUnset("global", profileKey(cmd.profile, f.field), f.value); err != nil {
			return err
		}
	}
	fmt.Fprintf(stdout, "saved profile %s: %s\n", cmd.profile, formatIdentity(cmd.userName, cmd.email))
	return nil
}

func remove(cmd command, stdout io.Writer) error {
	if _, err := findProfile(cmd.profile); err != nil {
		return err
	}
	if err := configRemoveSection("global", "profile."+cmd.profile); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "removed profile %s\n", cmd.profile)
	return nil
}

func currentIdentity() (name, email string, err error) {
	if name, err = configGet("", "user.name"); err != nil {
		return "", "", err
	}
	if email, err = configGet("", "user.email"); err != nil {
		return "", "", err
	}
	return name, email, nil
}

func formatIdentity(name, email string) string {
	return fmt.Sprintf("%s <%s>", name, email)
}
