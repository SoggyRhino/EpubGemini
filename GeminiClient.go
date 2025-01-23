package main

import (
	"context"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"log"
	"math"
	"time"
)

type GeminiClient struct {
	ctx         context.Context
	client      *genai.Client
	model       *genai.GenerativeModel
	instruction string
	prompt      string
	slot        time.Duration

	Input  chan Chapter
	Output chan Result
}
type Chapter struct {
	Filename string
	Content  string
	Context  string
}

type Result struct {
	Filename string
	Request  string
	Response string
	Error    error
	Tokens   int
	Duration time.Duration
	Chap     Chapter
}

func NewGeminiClient(modelName, key, instruction, prompt string) (*GeminiClient, error) {

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(key))
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}

	model := client.GenerativeModel(modelName)

	// Configure model parameters
	model.SetTemperature(0.3) // Lower temperature for more focused output
	model.SetTopP(0.8)        // Balanced diversity

	model.SystemInstruction = genai.NewUserContent(genai.Text(instruction))

	return &GeminiClient{
		ctx:         ctx,
		client:      client,
		model:       model,
		instruction: instruction,
		prompt:      prompt,
		slot:        rates[modelName],
		Input:       make(chan Chapter, 5000),
		Output:      make(chan Result, 100),
	}, nil
}

func (gc *GeminiClient) Start() {
	for chapter := range gc.Input {
		result := Result{
			Filename: chapter.Filename,
			Request:  gc.prompt + "\n" + chapter.Content + "\n" + chapter.Context,
			Response: "",
			Error:    nil,
			Tokens:   0,
			Duration: 0,
			Chap:     chapter,
		}

		if result.Tokens = len(result.Request) / 4; result.Tokens > 1_000_000 {
			result.Error = fmt.Errorf("request exceeds maximum token limit of 1M tokens")
			continue
		}

		// Calculate how many "slots" this request will use based on tokens
		slotsNeeded := int(math.Ceil(float64(result.Tokens) / (1_000_000.0 / 15.0)))
		timeout := gc.slot * time.Duration(slotsNeeded)

		go func(gc *GeminiClient, res *Result) { // Change to pointer
			start := time.Now()
			res.Response, res.Error = gc.processRequest(genai.Text(res.Request))
			res.Duration = time.Since(start)
			gc.Output <- *res
		}(gc, &result)

		time.Sleep(timeout)

	}

}

func (gc *GeminiClient) processRequest(text genai.Text) (string, error) {
	resp, err := gc.model.GenerateContent(gc.ctx, text)
	if err != nil {
		return "", fmt.Errorf("Error Generating Content: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return "", fmt.Errorf("No Response Generated")
	}

	return string(resp.Candidates[0].Content.Parts[0].(genai.Text)), nil
}

/*
	func (gc *GeminiClient) ExecuteRequestsContext(before int, after int) {
		results := make([]Result, 0, len(gc.I))
		for i, chapter := range gc.chapters {
			var builder strings2.Builder
			builder.WriteString(gc.prompt)
			builder.WriteString(chapter.Content)

			builder.WriteString("Chapters Before: \n")
			for j := int(math.Max(0, float64(i-before))); j < i; j++ {
				builder.WriteString(gc.chapters[j].Content)
			}

			builder.WriteString("Chapters After: \n")
			for j := int(math.Min(float64(len(gc.chapters)), float64(i+after))); j < i; j++ {
				builder.WriteString(gc.chapters[j].Content)
			}

			results = append(results, Result{
				Filename: chapter.Filename,
				Request:  builder.String(),
			})
		}
		gc.execute(results)
	}

	func (gc *GeminiClient) execute(results []Result) {
		var timeout time.Duration
		for _, result := range results {
			if result.Tokens = len(result.Request) / 4; result.Tokens > 1_000_000 {
				result.Error = fmt.Errorf("request exceeds maximum token limit of 1M tokens")
				continue
			}
			// Calculate how many "slots" this request will use based on tokens
			slotsNeeded := int(math.Ceil(float64(result.Tokens) / (1_000_000.0 / 15.0)))
			timeout = slot * time.Duration(slotsNeeded)

			go func(gc *GeminiClient, res Result) {
				start := time.Now()
				res.Response, res.Error = gc.tryNTimes(genai.Text(res.Request), 3, timeout)
				res.Duration = time.Since(start)
				gc.Output <- res
			}(gc, result)

			time.Sleep(timeout)
		}
	}

	func (gc *GeminiClient) processRequest(text genai.Text) (string, error) {
		resp, err := gc.model.GenerateContent(gc.ctx, text)
		if err != nil {
			return "", fmt.Errorf("error generating content: %w", err)
		}

		if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
			return "", fmt.Errorf("no response generated")
		}

		return string(resp.Candidates[0].Content.Parts[0].(genai.Text)), nil
	}

	func (gc *GeminiClient) tryNTimes(text genai.Text, n int, timeout time.Duration) (string, error) {
		if n == 0 {
			return "", fmt.Errorf("Request Faild Too Many Times\n")
		}
		request, err := gc.processRequest(text)
		time.Sleep(timeout)

		if err != nil {
			log.Printf("Error processing request: %v\nRetrying...\n", err)
			return gc.tryNTimes(text, n-1, timeout)
		}

		if len(request) == 0 {
			log.Printf("Response was Empty\nRetrying...\n", err)
			return gc.tryNTimes(text, n-1, timeout)
		}

		return request, nil
	}
*/
func (gc *GeminiClient) Close() {
	if gc.client != nil {
		close(gc.Input)
		close(gc.Output)
		err := gc.client.Close()
		if err != nil {
			log.Printf("error closing gemini client: %v", err)
		}
	}
}
