package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var rates = map[string]time.Duration{
	"gemini-1.5-pro":      time.Minute / 15,
	"gemini-1.5-flash":    time.Minute / 15,
	"gemini-1.5-flash-8b": time.Minute / 2,
}

type Args struct {
	File          string `json:"file"`
	Directory     string `json:"directory"`
	ContextBefore int    `json:"contextBefore"`
	ContextAfter  int    `json:"contextAfter"`
	APIKey        string `json:"APIKey"`
	Prompt        string `json:"prompt"`
	Instruction   string `json:"instruction"`
	Model         string `json:"model"`
	RateLimit     time.Duration
}

func validateEpubFile(path string) error {
	if !strings.HasSuffix(strings.ToLower(path), ".epub") {
		return errors.New("input file must be an .epub file")
	}
	return nil
}

func parseArgs() (string, Args, error) {
	// Define action flags
	helpFlag := flag.Bool("help", false, "Display help information")
	jsonFlag := flag.String("j", "", "Load arguments from JSON file")

	// Define Args flags
	fileFlag := flag.String("f", "", "Input epub file path (required)")
	dirFlag := flag.String("d", "output", "Output directory")
	contextBeforeFlag := flag.Int("cb", 0, "Context before")
	contextAfterFlag := flag.Int("ca", 0, "Context after")
	apiKeyFlag := flag.String("key", "", "API Key (required)")
	promptFlag := flag.String("prompt", "", "Prompt (required)")
	instructionFlag := flag.String("instruction", "", "Instruction (required)")
	modelFlag := flag.String("model", "", "Model name (required)")

	flag.Parse()

	// Check for help action first
	if *helpFlag {
		return "help", Args{}, nil
	}

	// Handle JSON input
	if *jsonFlag != "" {
		// When using -j, no other flags should be present
		if *fileFlag != "" || *apiKeyFlag != "" || *promptFlag != "" ||
			*instructionFlag != "" || *modelFlag != "" {
			return "run", Args{}, errors.New("when using -j, no other flags should be provided")
		}

		var args Args
		data, err := os.ReadFile(*jsonFlag)
		if err != nil {
			return "run", Args{}, errors.New("failed to read JSON file: " + err.Error())
		}

		if err := json.Unmarshal(data, &args); err != nil {
			return "run", Args{}, errors.New("failed to parse JSON file: " + err.Error())
		}

		// Validate epub file
		if err := validateEpubFile(args.File); err != nil {
			return "run", Args{}, err
		}

		// Set rate limit based on model
		if rateLimit, ok := rates[args.Model]; ok {
			args.RateLimit = rateLimit
		} else {
			return "run", Args{}, fmt.Errorf("model %s is not a valid model", args.Model)
		}

		return "run", args, nil
	}

	// Validate required flags when not using JSON
	if *fileFlag == "" {
		return "run", Args{}, errors.New("file (-f) is required")
	}
	if *apiKeyFlag == "" {
		return "run", Args{}, errors.New("API key (-key) is required")
	}
	if *promptFlag == "" {
		return "run", Args{}, errors.New("prompt (-prompt) is required")
	}
	if *instructionFlag == "" {
		return "run", Args{}, errors.New("instruction (-instruction) is required")
	}
	if *modelFlag == "" {
		return "run", Args{}, errors.New("model (-model) is required")
	}

	// Validate epub file
	if err := validateEpubFile(*fileFlag); err != nil {
		return "run", Args{}, err
	}

	// Construct Args from flags
	args := Args{
		File:          *fileFlag,
		Directory:     filepath.Clean(*dirFlag),
		ContextBefore: *contextBeforeFlag,
		ContextAfter:  *contextAfterFlag,
		APIKey:        *apiKeyFlag,
		Prompt:        *promptFlag,
		Instruction:   *instructionFlag,
		Model:         *modelFlag,
	}

	// Set rate limit based on model
	if _, ok := rates[args.Model]; !ok {
		return "run", Args{}, fmt.Errorf("model %s is not a valid model", args.Model)
	}

	return "run", args, nil
}

func help() {
	fmt.Println(`
Usage: program [action] [options]

Actions:
    -help           Display this help message
    -j <file>       Load arguments from JSON file

Required Options (when not using -j):
    -f             Input epub file path
    -key           API Key
    -prompt        Prompt text
    -instruction   Instruction text
    -model         Model name (available: gemini-1.5-pro, gemini-1.5-flash, gemini-1.5-flash-8b)

Optional Options:
    -d             Output directory (default: "output")
    -cb            Context before (default: 0)
    -ca            Context after (default: 0)

Examples:
    # Using command line arguments:
    program -f input.epub -key YOUR_API_KEY -prompt "your prompt" -instruction "your instruction" -model "gemini-1.5-pro"

    # Using JSON file:
    program -j args.json

    # With optional parameters:
    program -f input.epub -d custom/output -cb 2 -ca 2 -key YOUR_API_KEY -prompt "prompt" -instruction "instruction" -model "gemini-1.5-pro"

JSON File Format:
    {
        "file": "input.epub",
        "directory": "output",
        "contextBefore": 0,
        "contextAfter": 0,
        "APIKey": "YOUR_API_KEY",
        "prompt": "your prompt",
        "instruction": "your instruction",
        "model": "gemini-1.5-pro"
    }`)
}
