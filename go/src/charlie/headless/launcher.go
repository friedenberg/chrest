package headless

import (
	"context"
	"regexp"
	"runtime"

	"code.linenisgreat.com/chrest/go/src/charlie/launcher"
)

func Launch(ctx context.Context) (*launcher.Process, error) {
	return launcher.Launch(ctx, chromeConfig())
}

func chromeConfig() launcher.BrowserConfig {
	cfg := launcher.BrowserConfig{
		BinaryNames: []string{"google-chrome", "google-chrome-stable", "chromium"},
		Args: []string{
			"--headless",
			"--disable-gpu",
			"--no-first-run",
			"--no-default-browser-check",
			"--disable-extensions",
		},
		URL: "about:blank",
	}

	if runtime.GOOS == "darwin" {
		cfg.BinaryNames = append([]string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		}, cfg.BinaryNames...)
		cfg.Discovery = launcher.DiscoveryStrategy{
			Kind: launcher.HTTPEndpoint,
		}
	} else {
		cfg.Args = append(cfg.Args, "--remote-debugging-port=0")
		cfg.Discovery = launcher.DiscoveryStrategy{
			Kind:    launcher.StderrRegex,
			Pattern: regexp.MustCompile(`DevTools listening on (ws://\S+)`),
		}
	}

	return cfg
}
