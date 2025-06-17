package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"chunker/internal/service"
	"chunker/internal/chunking"
)

func TestCLIRunner_Run(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		flags    CLIFlags
		wantErr  bool
		contains []string
	}{
		{
			name:  "simple JSON output",
			input: "Hello world. This is a test.",
			flags: CLIFlags{
				ChunkSize:    20,
				Overlap:      5,
				OutputFormat: "json",
			},
			wantErr:  false,
			contains: []string{`"total_chunks":2`, `"text":"Hello world."`, `"text":"This is a test."`},
		},
		{
			name:  "JSONL output format",
			input: "Hello world. This is a test.",
			flags: CLIFlags{
				ChunkSize:    20,
				Overlap:      5,
				OutputFormat: "jsonl",
			},
			wantErr:  false,
			contains: []string{`"type":"metadata"`, `"type":"chunk"`, `"text":"Hello world."`},
		},
		{
			name:  "empty input error",
			input: "",
			flags: CLIFlags{
				ChunkSize: 20,
			},
			wantErr: true,
		},
		{
			name:  "word boundary strategy",
			input: "One two three four five six seven eight nine ten",
			flags: CLIFlags{
				ChunkSize: 15,
				Overlap:   5,
				Strategy:  "word_boundary",
			},
			wantErr:  false,
			contains: []string{`"strategy_used":"word_boundary"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			factory := chunking.NewFactory()
			chunkService := service.NewChunkService(factory)
			
			input := strings.NewReader(tt.input)
			output := &bytes.Buffer{}
			errOutput := &bytes.Buffer{}
			
			runner := NewCLIRunner(chunkService, input, output, errOutput)
			
			// Execute
			err := runner.Run(context.Background(), tt.flags)
			
			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			if !tt.wantErr {
				outputStr := output.String()
				for _, want := range tt.contains {
					if !strings.Contains(outputStr, want) {
						t.Errorf("Output missing expected content: %q\nGot: %s", want, outputStr)
					}
				}
			}
		})
	}
}

func TestCLIRunner_OutputFormats(t *testing.T) {
	factory := chunking.NewFactory()
	chunkService := service.NewChunkService(factory)
	
	input := strings.NewReader("Test content for output format testing.")
	
	t.Run("pretty JSON", func(t *testing.T) {
		output := &bytes.Buffer{}
		runner := NewCLIRunner(chunkService, input, output, &bytes.Buffer{})
		
		flags := CLIFlags{
			ChunkSize:    100,
			OutputFormat: "json",
			Pretty:       true,
		}
		
		err := runner.Run(context.Background(), flags)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		
		// Pretty JSON should have indentation
		if !strings.Contains(output.String(), "\n  ") {
			t.Error("Expected pretty-printed JSON with indentation")
		}
	})
}