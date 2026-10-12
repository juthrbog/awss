package browser

import (
	"errors"
	"testing"
	"time"

	pkgbrowser "github.com/pkg/browser"
)

const testURL = "https://example.awsapps.com/start"

// stubOpener replaces the opener command and handoffWait for one test.
func stubOpener(t *testing.T, wait time.Duration, open func(string) error) {
	t.Helper()
	oldOpen, oldWait := openURL, handoffWait
	openURL, handoffWait = open, wait
	t.Cleanup(func() { openURL, handoffWait = oldOpen, oldWait })
}

func TestOpen_OpenerHandsOff(t *testing.T) {
	var got string
	stubOpener(t, time.Minute, func(url string) error {
		got = url
		return nil
	})

	if err := Open(testURL); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if got != testURL {
		t.Errorf("opener got %q, want %q", got, testURL)
	}
}

func TestOpen_ReturnsOpenerError(t *testing.T) {
	want := errors.New("no browser found")
	stubOpener(t, time.Minute, func(string) error { return want })

	if err := Open(testURL); !errors.Is(err, want) {
		t.Errorf("Open() error = %v, want %v", err, want)
	}
}

// When no browser runs, xdg-open starts one and runs until the browser exits.
// Open must return anyway, so that the login can poll for the token.
func TestOpen_DoesNotWaitForNewBrowser(t *testing.T) {
	browserExit := make(chan struct{})
	t.Cleanup(func() { close(browserExit) })
	stubOpener(t, 10*time.Millisecond, func(string) error {
		<-browserExit
		return nil
	})

	done := make(chan error, 1)
	go func() { done <- Open(testURL) }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Open() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Open() still waits for the browser to exit")
	}
}

func TestBrowserOutputGoesToNullDevice(t *testing.T) {
	if pkgbrowser.Stdout != nil || pkgbrowser.Stderr != nil {
		t.Errorf("browser output goes to %v and %v, want the null device (nil)", pkgbrowser.Stdout, pkgbrowser.Stderr)
	}
}
