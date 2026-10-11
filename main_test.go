package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		args    []string
		want    command
		wantErr string // substring of the expected usage error
	}{
		{args: nil, want: command{action: "show"}},
		{args: []string{"-h"}, want: command{action: "help"}},
		{args: []string{"--help"}, want: command{action: "help"}},
		{args: []string{"help"}, want: command{action: "help"}},
		{args: []string{"add", "frank", "-h"}, want: command{action: "help"}},
		{args: []string{"list"}, want: command{action: "list"}},
		{args: []string{"list", "x"}, wantErr: "no arguments"},

		{args: []string{"frank"}, want: command{action: "apply", profile: "frank"}},
		{args: []string{"frank", "--global"}, want: command{action: "apply", profile: "frank"}},
		{args: []string{"frank", "--local"}, want: command{action: "apply", profile: "frank", local: true}},
		{args: []string{"--local", "frank"}, want: command{action: "apply", profile: "frank", local: true}},
		{args: []string{"frank", "--local", "--global"}, wantErr: "not both"},
		{args: []string{"--local"}, wantErr: "missing profile name"},
		{args: []string{"--global"}, wantErr: "missing profile name"},
		{args: []string{"frank", "you"}, wantErr: "too many arguments"},
		{args: []string{"frank", "--global=yes"}, wantErr: "takes no value"},
		{args: []string{"frank", "--repo"}, wantErr: "unknown flag --repo"},
		{args: []string{"frank", "-g"}, wantErr: "unknown flag -g"},
		{args: []string{"bad name"}, wantErr: "invalid profile name"},

		{
			args: []string{"add", "frank", "--name", "Frank Reynolds", "--email=frank@example.com"},
			want: command{action: "add", profile: "frank", userName: "Frank Reynolds", email: "frank@example.com"},
		},
		{
			args: []string{"add", "--name", "Mantis Toboggan", "mantis", "--email", "mantis@example.com",
				"--signingkey", "ABC123", "--sshcommand", "ssh -i ~/.ssh/mantis"},
			want: command{action: "add", profile: "mantis", userName: "Mantis Toboggan", email: "mantis@example.com",
				signingKey: "ABC123", sshCommand: "ssh -i ~/.ssh/mantis"},
		},
		{args: []string{"add", "frank", "--email", "frank@example.com"}, wantErr: "--name is required"},
		{args: []string{"add", "frank", "--name", "Frank"}, wantErr: "--email is required"},
		{args: []string{"add", "frank", "--name", "", "--email", "e"}, wantErr: "--name is required"},
		{args: []string{"add", "frank", "--name"}, wantErr: "--name needs a value"},
		{args: []string{"add", "frank", "--global"}, wantErr: "unknown flag --global"},
		{args: []string{"add", "--name", "x", "--email", "y"}, wantErr: "missing profile name"},
		{args: []string{"add", "list", "--name", "x", "--email", "y"}, wantErr: "reserved"},
		{args: []string{"add", "help", "--name", "x", "--email", "y"}, wantErr: "reserved"},

		{args: []string{"remove", "frank"}, want: command{action: "remove", profile: "frank"}},
		{args: []string{"remove"}, wantErr: "missing profile name"},
		{args: []string{"remove", "frank", "--global"}, wantErr: "unknown flag"},
	}
	for _, tt := range tests {
		got, err := parseArgs(tt.args)
		if tt.wantErr != "" {
			var uerr usageError
			if !errors.As(err, &uerr) || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("parseArgs(%q) error = %v, want usage error containing %q", tt.args, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseArgs(%q) unexpected error: %v", tt.args, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parseArgs(%q) = %+v, want %+v", tt.args, got, tt.want)
		}
	}
}
