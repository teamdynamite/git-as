package main

import (
	"reflect"
	"testing"
)

func TestValidateProfileName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"frank", false},
		{"mantis", false},
		{"work-2", false},
		{"my.work", false},
		{"_bot", false},
		{"Frank", false},
		{"", true},
		{"add", true},
		{"remove", true},
		{"list", true},
		{"help", true},
		{"-x", true},
		{".hidden", true},
		{"has space", true},
		{"a/b", true},
		{"ünicode", true},
	}
	for _, tt := range tests {
		err := validateProfileName(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateProfileName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
		}
	}
}

func TestParseProfiles(t *testing.T) {
	tests := []struct {
		desc string
		in   string
		want []profile
	}{
		{"empty", "", []profile{}},
		{
			"two profiles sorted",
			"profile.frank.name\nFrank Reynolds\x00profile.frank.email\nfrank@example.com\x00" +
				"profile.mantis.name\nMantis Toboggan\x00profile.mantis.email\nmantis@example.com\x00" +
				"profile.mantis.signingkey\nABC\x00profile.mantis.sshcommand\nssh -i k\x00",
			[]profile{
				{name: "frank", userName: "Frank Reynolds", email: "frank@example.com"},
				{name: "mantis", userName: "Mantis Toboggan", email: "mantis@example.com", signingKey: "ABC", sshCommand: "ssh -i k"},
			},
		},
		{
			"dotted subsection",
			"profile.my.work.email\nw@example.com\x00",
			[]profile{{name: "my.work", email: "w@example.com"}},
		},
		{
			"value with newline",
			"profile.frank.name\nline1\nline2\x00",
			[]profile{{name: "frank", userName: "line1\nline2"}},
		},
		{
			"no subsection and unknown fields ignored",
			"profile.loose\nx\x00profile.frank.other\ny\x00",
			[]profile{{name: "frank"}},
		},
	}
	for _, tt := range tests {
		got := parseProfiles(tt.in)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: parseProfiles() = %+v, want %+v", tt.desc, got, tt.want)
		}
	}
}
