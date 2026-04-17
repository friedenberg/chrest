package firefox

import (
	"context"
	"regexp"

	"code.linenisgreat.com/chrest/go/src/charlie/launcher"
)

func Launch(ctx context.Context) (*launcher.Process, error) {
	return launcher.Launch(ctx, firefoxConfig())
}

func firefoxConfig() launcher.BrowserConfig {
	return launcher.BrowserConfig{
		BinaryNames: []string{"firefox", "firefox-esr"},
		Args: []string{
			"--headless",
			"--remote-debugging-port=0",
			"--no-remote",
		},
		URL:         "about:blank",
		TempProfile: "chrest-firefox-",
		Discovery: launcher.DiscoveryStrategy{
			Kind:    launcher.StderrRegex,
			Pattern: regexp.MustCompile(`WebDriver BiDi listening on (ws://\S+)`),
		},
	}
}
