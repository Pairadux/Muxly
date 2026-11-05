package tmuxtest

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestWrapperReachesTheServerAndLogsCalls(t *testing.T) {
	bin := t.TempDir()
	srv := New(t, bin)
	srv.NewSession("a")

	out, err := exec.Command(bin+"/tmux", "-u", "list-sessions", "-F", "#{session_name}").Output()
	if err != nil {
		t.Fatal(err)
	}

	if got := strings.TrimSpace(string(out)); got != "a" {
		t.Errorf("wrapper listed %q, want the private session", got)
	}
	if got := srv.Calls(); len(got) != 1 || !strings.Contains(got[0], "list-sessions") {
		t.Errorf("Calls() = %q, want only the wrapper's call", got)
	}
}

func TestFailOnRefusesOneSubcommand(t *testing.T) {
	bin := t.TempDir()
	srv := New(t, bin)
	srv.NewSession("a")
	srv.FailOn("kill-session")

	out, err := exec.Command(bin+"/tmux", "-u", "kill-session", "-t", "a").CombinedOutput()

	if err == nil || !strings.Contains(string(out), InjectedFailure) {
		t.Errorf("err = %v, out = %q: want the injected failure", err, out)
	}
	if got := srv.Sessions(); !slices.Equal(got, []string{"a"}) {
		t.Errorf("sessions = %q: the failed call should never have reached tmux", got)
	}
}

func TestAttachGivesARealClient(t *testing.T) {
	srv := New(t, t.TempDir())
	id := srv.NewSession("a")

	srv.Attach(id)

	if got := srv.ClientSession(); got != "a" {
		t.Errorf("client is on %q, want a", got)
	}
}
