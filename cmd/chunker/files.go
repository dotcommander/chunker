package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dotcommander/chunker/internal/domain"
)

func runFilesCommand(args []string) {
	filesFlags := newChunkFlagSet("files")
	outDir := filesFlags.String("out-dir", "chunks", "Output directory")
	if err := filesFlags.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if filesFlags.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "files command requires a glob pattern, e.g. chunker files \"docs/*.md\"")
		os.Exit(1)
	}

	pattern := filesFlags.Arg(0)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid glob pattern: %v\n", err)
		os.Exit(1)
	}
	if len(matches) == 0 {
		fmt.Fprintf(os.Stderr, "no files matched pattern: %s\n", pattern)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	chunkService := newChunkService()
	processed, failures := processBatchFiles(chunkService, matches, *outDir, os.Stderr)

	total := processed + len(failures)
	fmt.Fprintf(os.Stdout, "Processed %d of %d file(s) into %s; %d failed\n", processed, total, *outDir, len(failures))
	if len(failures) > 0 {
		fmt.Fprintln(os.Stdout, "Failures:")
		for _, f := range failures {
			fmt.Fprintf(os.Stdout, "  %s: %v\n", f.path, f.err)
		}
	}
}

// batchFailure pairs a source file path with the error that aborted its
// processing, so the per-file summary can be emitted at the end of a batch run.
type batchFailure struct {
	path string
	err  error
}

// readBoundedFile reads up to limit bytes from the file at path, returning
// ErrStdinTooLarge when the file exceeds the cap. Mirrors the stdin and HTTP
// body caps so a large glob match cannot exhaust process memory.
func readBoundedFile(path string, limit int64) (data []byte, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close input file: %w", closeErr))
		}
	}()
	return readAllBounded(f, limit)
}

// processBatchFiles runs the per-file chunk-and-write pipeline over matches,
// returning the success count and the per-file failure list. Directories and
// stat-failures are skipped silently; all other failures are recorded.
// Caller owns stderr formatting of per-file skip lines.
func processBatchFiles(svc domain.ChunkService, matches []string, outDir string, stderr io.Writer) (int, []batchFailure) {
	processed := 0
	var failures []batchFailure
	for _, filePath := range matches {
		info, statErr := os.Stat(filePath)
		if statErr != nil || info.IsDir() {
			continue
		}

		content, readErr := readBoundedFile(filePath, stdinLimit())
		if readErr != nil {
			fmt.Fprintf(stderr, "skip %s: %v\n", filePath, readErr)
			failures = append(failures, batchFailure{filePath, readErr})
			continue
		}

		req := chunkRequestFromFlags(string(content))
		resp, procErr := svc.ProcessChunkRequest(context.Background(), req)
		if procErr != nil {
			fmt.Fprintf(stderr, "skip %s: %v\n", filePath, procErr)
			failures = append(failures, batchFailure{filePath, procErr})
			continue
		}

		if writeErr := writeFileOutput(svc, outDir, filePath, resp); writeErr != nil {
			fmt.Fprintf(stderr, "skip %s: %v\n", filePath, writeErr)
			failures = append(failures, batchFailure{filePath, writeErr})
			continue
		}

		processed++
	}
	return processed, failures
}

func writeFileOutput(chunkService domain.ChunkService, outDir string, sourcePath string, resp *domain.ChunkResponse) error {
	base := filepath.Base(sourcePath)
	base = strings.TrimSuffix(base, filepath.Ext(base))

	var ext string
	if *outputFormat == "jsonl" {
		ext = ".jsonl"
	} else {
		ext = ".json"
	}

	outPath := filepath.Join(outDir, base+ext)
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	return writeOutputAndClose(chunkService, resp, f)
}

func writeOutputAndClose(chunkService domain.ChunkService, resp *domain.ChunkResponse, output io.WriteCloser) error {
	runner := NewCLIRunner(chunkService, os.Stdin, output, os.Stderr)
	writeErr := runner.writeOutput(resp, *outputFormat, *pretty)
	closeErr := output.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("close output file: %w", closeErr)
	}
	return errors.Join(writeErr, closeErr)
}
