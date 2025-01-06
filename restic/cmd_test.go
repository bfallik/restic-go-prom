package restic

import (
	"os/exec"
	"strings"
	"testing"
)

func TestCmdVersion(t *testing.T) {
	c, err := NewCmd(CmdOptCommandVersion{}, CmdOptFlags{"--json"})
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Run(); err != nil {
		t.Fatal(err)
	}

	cnt := strings.Count(c.OutBuf.String(), "version")
	if cnt != 2 {
		t.Errorf("unexpected \"version\" substrings(expected 2, actual %d): %s", cnt, c.OutBuf.String())
	}
}

func TestCmdError(t *testing.T) {
	c, err := NewCmd(CmdOptCommandVersion{}, CmdOptFlags{"--not-a-valid-flag"})
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Run(); err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if exiterr.ExitCode() != 1 {
				t.Errorf("unexpected exit code: %d", exiterr.ExitCode())
			}
		} else {
			t.Fatalf("Cmd.Run: %v", err)
		}
	}
}
