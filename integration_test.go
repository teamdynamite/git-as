package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binDir holds a built git-as binary, for the test that runs it through git.
var binDir string

func TestMain(m *testing.M) {
	// Build before any test changes HOME, so the build uses the normal Go cache.
	dir, err := os.MkdirTemp("", "git-as-bin")
	if err != nil {
		panic(err)
	}
	build := exec.Command("go", "build", "-o", filepath.Join(dir, "git-as"), ".")
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic(err)
	}
	binDir = dir
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// setupEnv isolates git from the real user config, creates a temp repo, and
// changes into it. It returns the temp HOME and the repo path.
func setupEnv(t *testing.T) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// These variables would point git at other config files or repos.
	for _, v := range []string{"GIT_CONFIG_GLOBAL", "GIT_CONFIG_SYSTEM", "GIT_CONFIG_PARAMETERS",
		"GIT_CONFIG_COUNT", "GIT_DIR", "GIT_WORK_TREE", "GIT_AUTHOR_NAME", "GIT_AUTHOR_EMAIL",
		"GIT_COMMITTER_NAME", "GIT_COMMITTER_EMAIL"} {
		t.Setenv(v, "")
		os.Unsetenv(v)
	}

	repo = filepath.Join(t.TempDir(), "repo")
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(repo))
	mustGit(t, "init", "-q", repo)
	t.Chdir(repo)
	return home, repo
}

// runAs runs git-as in-process and returns its output and exit code.
func runAs(t *testing.T, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return out.String(), errOut.String(), code
}

// mustRunAs runs git-as and fails the test if it does not exit 0.
func mustRunAs(t *testing.T, args ...string) string {
	t.Helper()
	stdout, stderr, code := runAs(t, args...)
	if code != 0 {
		t.Fatalf("git as %q: exit %d, stderr: %s", args, code, stderr)
	}
	return stdout
}

func mustGit(t *testing.T, args ...string) string {
	t.Helper()
	out, err := git(args...)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(out)
}

// configValue returns a config value at a scope, or "" if it is not set.
func configValue(t *testing.T, scope, key string) string {
	t.Helper()
	v, err := configGet(scope, key)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func addProfiles(t *testing.T) {
	t.Helper()
	mustRunAs(t, "add", "frank", "--name", "Frank Reynolds", "--email", "frank@example.com",
		"--signingkey", "ABC123", "--sshcommand", "ssh -i ~/.ssh/frank")
	mustRunAs(t, "add", "mantis", "--name", "Mantis Toboggan", "--email", "mantis@example.com")
}

func TestConfigStaysInTempHome(t *testing.T) {
	home, _ := setupEnv(t)
	addProfiles(t)
	data, err := os.ReadFile(filepath.Join(home, ".gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `[profile "frank"]`) {
		t.Errorf("temp .gitconfig does not contain the profile:\n%s", data)
	}
}

func TestShowAndList(t *testing.T) {
	setupEnv(t)

	out := mustRunAs(t)
	if !strings.Contains(out, "no identity set") || !strings.Contains(out, "no profiles") {
		t.Errorf("show with empty config:\n%s", out)
	}

	addProfiles(t)
	out = mustRunAs(t, "list")
	want := "  frank   Frank Reynolds <frank@example.com>\n  mantis  Mantis Toboggan <mantis@example.com>\n"
	if out != want {
		t.Errorf("list:\n%s\nwant:\n%s", out, want)
	}

	mustRunAs(t, "frank")
	out = mustRunAs(t)
	if !strings.Contains(out, "current: Frank Reynolds <frank@example.com> (profile: frank)") {
		t.Errorf("show does not name the matching profile:\n%s", out)
	}
	if !strings.Contains(out, "* frank ") {
		t.Errorf("show does not mark the matching profile:\n%s", out)
	}
}

func TestApplyLocal(t *testing.T) {
	setupEnv(t)
	addProfiles(t)

	mustRunAs(t, "frank", "--local")
	for key, want := range map[string]string{
		"user.name":       "Frank Reynolds",
		"user.email":      "frank@example.com",
		"user.signingkey": "ABC123",
		"core.sshCommand": "ssh -i ~/.ssh/frank",
	} {
		if got := configValue(t, "local", key); got != want {
			t.Errorf("local %s = %q, want %q", key, got, want)
		}
	}
	if got := configValue(t, "global", "user.name"); got != "" {
		t.Errorf("global user.name = %q, want unset", got)
	}

	mustGit(t, "commit", "-q", "--allow-empty", "-m", "test")
	if got := mustGit(t, "log", "-1", "--format=%an <%ae>"); got != "Frank Reynolds <frank@example.com>" {
		t.Errorf("commit author = %q", got)
	}
}

func TestApplyUnsetsOptionalValues(t *testing.T) {
	setupEnv(t)
	addProfiles(t)

	mustRunAs(t, "frank", "--local")
	mustRunAs(t, "mantis", "--local")
	if got := configValue(t, "local", "user.email"); got != "mantis@example.com" {
		t.Errorf("local user.email = %q", got)
	}
	for _, key := range []string{"user.signingkey", "core.sshCommand"} {
		if got := configValue(t, "local", key); got != "" {
			t.Errorf("local %s = %q, want unset", key, got)
		}
	}
}

func TestApplyGlobal(t *testing.T) {
	setupEnv(t)
	addProfiles(t)

	_, stderr, code := runAs(t, "mantis")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("unexpected stderr: %s", stderr)
	}
	if got := configValue(t, "global", "user.email"); got != "mantis@example.com" {
		t.Errorf("global user.email = %q", got)
	}
	if got := configValue(t, "local", "user.email"); got != "" {
		t.Errorf("local user.email = %q, want unset", got)
	}

	mustRunAs(t, "frank", "--global")
	if got := configValue(t, "global", "user.email"); got != "frank@example.com" {
		t.Errorf("global user.email after --global = %q", got)
	}

	// A local identity overrides the global one, so git as warns about it.
	mustRunAs(t, "frank", "--local")
	_, stderr, _ = runAs(t, "mantis")
	if !strings.Contains(stderr, "overrides the global one") {
		t.Errorf("no override note, stderr: %q", stderr)
	}
}

func TestApplyOutsideRepo(t *testing.T) {
	setupEnv(t)
	addProfiles(t)
	t.Chdir(t.TempDir())

	mustRunAs(t, "frank")
	if got := configValue(t, "global", "user.name"); got != "Frank Reynolds" {
		t.Errorf("global user.name = %q", got)
	}
}

func TestAddReplacesProfile(t *testing.T) {
	setupEnv(t)
	addProfiles(t)

	mustRunAs(t, "add", "frank", "--name", "Frank", "--email", "frank@new.example.com")
	p, err := findProfile("frank")
	if err != nil {
		t.Fatal(err)
	}
	want := profile{name: "frank", userName: "Frank", email: "frank@new.example.com"}
	if p != want {
		t.Errorf("profile = %+v, want %+v", p, want)
	}
}

func TestRemove(t *testing.T) {
	setupEnv(t)
	addProfiles(t)

	mustRunAs(t, "remove", "mantis")
	out := mustRunAs(t, "list")
	if strings.Contains(out, "mantis") {
		t.Errorf("mantis still listed:\n%s", out)
	}
	if !strings.Contains(out, "frank") {
		t.Errorf("frank missing:\n%s", out)
	}
}

func TestErrors(t *testing.T) {
	setupEnv(t)
	addProfiles(t)
	notRepo := t.TempDir()

	tests := []struct {
		desc     string
		dir      string // run in this directory; "" means the repo
		args     []string
		wantCode int
		wantErr  string
	}{
		{"unknown profile", "", []string{"nobody"}, 1, `no profile named "nobody"`},
		{"remove unknown profile", "", []string{"remove", "nobody"}, 1, `no profile named "nobody"`},
		{"--local outside repo", notRepo, []string{"frank", "--local"}, 1, "not inside a git repository"},
		{"reserved name", "", []string{"add", "remove", "--name", "x", "--email", "y"}, 2, "reserved"},
		{"unknown flag", "", []string{"frank", "--bogus"}, 2, "unknown flag --bogus"},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if tt.dir != "" {
				t.Chdir(tt.dir)
			}
			stdout, stderr, code := runAs(t, tt.args...)
			if code != tt.wantCode {
				t.Errorf("exit %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tt.wantErr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
		})
	}
}

func TestRunsAsGitSubcommand(t *testing.T) {
	setupEnv(t)
	addProfiles(t)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	out := mustGit(t, "as", "frank")
	if !strings.Contains(out, "now using frank") {
		t.Errorf("git as frank: %q", out)
	}
	if got := configValue(t, "global", "user.name"); got != "Frank Reynolds" {
		t.Errorf("global user.name = %q", got)
	}
}
