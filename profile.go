package main

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// reservedNames are subcommands, so they cannot be profile names.
var reservedNames = []string{"add", "remove", "list", "help"}

var profileNameRE = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)

// profile is one identity stored in the global config as profile.<name>.*.
type profile struct {
	name       string // profile name, e.g. "frank"
	userName   string
	email      string
	signingKey string
	sshCommand string
}

func validateProfileName(name string) error {
	switch {
	case name == "":
		return errors.New("profile name is empty")
	case slices.Contains(reservedNames, name):
		return fmt.Errorf("%q is reserved and cannot be a profile name", name)
	case !profileNameRE.MatchString(name):
		return fmt.Errorf("invalid profile name %q: use letters, digits, '.', '_' and '-'", name)
	}
	return nil
}

func profileKey(name, field string) string {
	return "profile." + name + "." + field
}

// loadProfiles reads all profiles from the global config, sorted by name.
func loadProfiles() ([]profile, error) {
	out, err := git("config", "--global", "-z", "--get-regexp", `^profile\.`)
	if exitCode(err) == exitKeyNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseProfiles(out), nil
}

// parseProfiles parses `git config -z --get-regexp` output, where each entry
// is "key\nvalue" followed by a NUL byte.
func parseProfiles(out string) []profile {
	byName := map[string]*profile{}
	for _, entry := range strings.Split(out, "\x00") {
		key, value, _ := strings.Cut(entry, "\n")
		rest, ok := strings.CutPrefix(key, "profile.")
		if !ok {
			continue
		}
		// The subsection may contain dots; the variable name cannot.
		i := strings.LastIndex(rest, ".")
		if i <= 0 {
			continue
		}
		name, field := rest[:i], rest[i+1:]
		p := byName[name]
		if p == nil {
			p = &profile{name: name}
			byName[name] = p
		}
		switch field {
		case "name":
			p.userName = value
		case "email":
			p.email = value
		case "signingkey":
			p.signingKey = value
		case "sshcommand":
			p.sshCommand = value
		}
	}

	profiles := make([]profile, 0, len(byName))
	for _, p := range byName {
		profiles = append(profiles, *p)
	}
	slices.SortFunc(profiles, func(a, b profile) int { return strings.Compare(a.name, b.name) })
	return profiles
}

// findProfile loads the profile called name from the global config.
func findProfile(name string) (profile, error) {
	profiles, err := loadProfiles()
	if err != nil {
		return profile{}, err
	}
	for _, p := range profiles {
		if p.name == name {
			return p, nil
		}
	}
	return profile{}, fmt.Errorf("no profile named %q (see 'git as list')", name)
}
