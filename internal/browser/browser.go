// Package browser opens a URL in the default browser without waiting for the
// browser to exit.
package browser

import (
	"time"

	pkgbrowser "github.com/pkg/browser"
)

// handoffWait is how long Open waits for the opener command to exit.
var handoffWait = time.Second

// openURL runs the opener command and waits for it. Tests replace it.
var openURL = pkgbrowser.OpenURL

func init() {
	// The browser writes its own log lines. Send them to the null device, not
	// to the terminal that runs awss.
	pkgbrowser.Stdout = nil
	pkgbrowser.Stderr = nil
}

// Open opens url in the default browser.
//
// The opener command (for example xdg-open on Linux) exits at once when it
// gives the URL to a browser that runs already. When no browser runs, the
// opener starts one and does not exit until that browser exits. Open does not
// wait for that: if the opener still runs after handoffWait, Open returns nil
// and the opener continues in the background.
//
// If the opener fails before handoffWait, Open returns its error.
func Open(url string) error {
	open, wait := openURL, handoffWait
	errc := make(chan error, 1)
	go func() { errc <- open(url) }()
	select {
	case err := <-errc:
		return err
	case <-time.After(wait):
		return nil
	}
}
