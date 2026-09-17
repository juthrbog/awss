package cmd

import "testing"

func TestReleaseVersion(t *testing.T) {
	previous := version
	version = "1.2.3"
	t.Cleanup(func() { version = previous })
	for _, flag := range []string{"--version", "-v"} {
		out, err := executeRoot(flag)
		if err != nil || out != "awss version 1.2.3\n" {
			t.Fatalf("%s = %q, %v", flag, out, err)
		}
	}
}
