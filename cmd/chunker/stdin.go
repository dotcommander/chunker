package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
)

// MaxStdinBytes is the default cap for stdin reads (100 MiB). It bounds every
// `io.ReadAll(stdin)`-style consumer so a malicious or accidental large pipe
// cannot exhaust process memory. Override with CHUNKER_MAX_STDIN_BYTES (decimal
// byte count); zero or negative values fall back to the default.
const MaxStdinBytes int64 = 100 * 1024 * 1024

// ErrStdinTooLarge signals that input exceeded the configured cap.
var ErrStdinTooLarge = errors.New("stdin input exceeds maximum allowed size")

// stdinLimit returns the active byte cap, honouring CHUNKER_MAX_STDIN_BYTES
// when it parses to a positive integer.
func stdinLimit() int64 {
	if v := os.Getenv("CHUNKER_MAX_STDIN_BYTES"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return MaxStdinBytes
}

// readAllBounded reads up to limit bytes from r and returns ErrStdinTooLarge
// if the input would have exceeded the cap. The "peek one extra byte" idiom
// (LimitReader at limit+1) distinguishes "exactly limit" (ok) from
// "more than limit" (truncated → error).
func readAllBounded(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w (limit: %d bytes; override with CHUNKER_MAX_STDIN_BYTES)", ErrStdinTooLarge, limit)
	}
	return data, nil
}
