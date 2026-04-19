package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"code.linenisgreat.com/chrest/go/src/delta/capturebatch"
)

// cmdCaptureBatch implements the `chrest capture-batch` subcommand per
// RFC 0001 (Web Capture Archive Protocol). It reads a single JSON
// document from stdin, runs every capture sequentially, spawns a fresh
// writer subprocess per artifact to obtain its content-addressed ID,
// and writes a single JSON result object to stdout.
//
// Unlike `chrest capture`, this command is entirely machine-driven:
// its contract is JSON-on-stdin / JSON-on-stdout, not flags.
func cmdCaptureBatch(ctx context.Context, version string) error {
	// Same SIGPIPE rationale as cmdCapture — an orchestrator closing its
	// read end of our stdout during error handling MUST NOT kill us
	// before `defer`red writer cleanup runs.
	signal.Ignore(syscall.SIGPIPE)

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read stdin: %w", err)
	}

	var input capturebatch.Input
	if err := json.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("parse batch input: %w", err)
	}
	if input.Schema != capturebatch.InputSchema {
		return fmt.Errorf("schema MUST be %q, got %q", capturebatch.InputSchema, input.Schema)
	}

	out, err := capturebatch.Run(ctx, input.Captures, capturebatch.Options{
		CapturerVersion: version,
		Writer:          input.Writer,
		URL:             input.URL,
		Defaults:        input.Defaults,
	})
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("write batch output: %w", err)
	}
	return nil
}
