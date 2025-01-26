package main

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	goepub "github.com/bmaupin/go-epub"
	"github.com/taylorskalyo/goreader/epub"
)

func main() {
	action, args, err := parseArgs()
	if err != nil {
		log.Fatal(err)
	} else if action == "help" {
		help()
	} else if action == "run" {
		run(args)
	} else {
		log.Fatalf("Unknown action: %s", action)
	}
}

func run(args Args) {
	epubReader, err := epub.OpenReader(args.File)
	if err != nil {
		log.Fatalf("Failed to open EPUB file: %v", err)
	}
	defer epubReader.Close()

	chapters, err := loadUnprocessedChapters(epubReader.Rootfiles[0], args.Directory, args.ContextBefore, args.ContextAfter)
	if err != nil {
		log.Fatalf("Failed to load unprocessed chapters: %v", err)
	}

	geminiClient, err := NewGeminiClient(args.Model, args.APIKey, args.Instruction, args.Prompt)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer geminiClient.Close()
	go geminiClient.Start()

	for _, chapter := range chapters {
		geminiClient.Input <- chapter
	}

	for i := 0; i < len(chapters); {
		result := <-geminiClient.Output

		if result.Error != nil {
			fmt.Printf("[%s] Failed to proccess %s : %v\nRetrying...\n", time.Now().Format("15:04:05"), result.Filename, result.Error)
			geminiClient.Input <- result.Chap

		} else {
			fmt.Printf("[%s] Processed %s in %.2fs\n", time.Now().Format("15:04:05"), result.Filename, result.Duration.Seconds())
			err := os.WriteFile(filepath.Join(args.Directory, result.Filename), []byte(result.Response[:]), 0666)
			if err != nil {
				log.Fatalf("Failed to write file: %v", err)
			}
			i++
		}
	}

	err = saveEpub(epubReader.Rootfiles[0], args.Directory)
	if err != nil {
		log.Fatalf("Failed to save Epub: %v", err)
	}
}

func loadChapterContent(itemRef epub.Itemref) (string, error) {
	itemReader, err := itemRef.Open()
	if err != nil {
		return "", fmt.Errorf("error opening item '%s': %w", itemRef.HREF, err)
	}
	defer func() {
		if closeErr := itemReader.Close(); closeErr != nil {
			log.Printf("Error closing item '%s': %v", itemRef.HREF, closeErr)
		}
	}()

	content, err := io.ReadAll(itemReader)
	if err != nil {
		return "", fmt.Errorf("Error reading item '%s': %w", itemRef.HREF, err)
	}
	return string(content), nil
}

func loadExistingChapterContents(dir string, files []os.DirEntry) map[string]string {
	contents := make(map[string]string)
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			log.Printf("Failed to read file %s: %v", file.Name(), err)
			continue
		}
		contents[file.Name()] = string(data)
	}
	return contents
}

func loadUnprocessedChapters(rootFile *epub.Rootfile, dir string, contextBefore, contextAfter int) ([]Chapter, error) {
	if err := ensure(dir); err != nil {
		return nil, fmt.Errorf("failed to ensure directory exists: %w", err)
	}

	readDir, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Load all chapter existing contents into memory first
	chapterContents := loadExistingChapterContents(dir, readDir)

	// Pre-allocate slice for chapters based on spine items
	indexedContents := make([]string, len(rootFile.Spine.Itemrefs))
	chapters := make([]Chapter, 0, len(rootFile.Spine.Itemrefs))

	// First pass: Load and index all chapter contents
	for i, itemRef := range rootFile.Spine.Itemrefs {
		if !isChapter(itemRef.HREF) {
			continue
		}

		content, ok := chapterContents[itemRef.HREF]
		if !ok {
			content, err = loadChapterContent(itemRef)
			if err != nil {
				log.Printf("Failed to load chapter content for %s: %v", itemRef.HREF, err)
				continue
			}
		}
		indexedContents[i] = content
	}

	// Second pass: Build chapters with context
	for i, itemRef := range rootFile.Spine.Itemrefs {
		if !isChapter(itemRef.HREF) {
			continue
		}

		context := buildChapterContext(indexedContents, i, contextBefore, contextAfter)

		chapter := Chapter{
			Filename: itemRef.HREF,
			Content:  indexedContents[i],
			Context:  context,
		}
		chapters = append(chapters, chapter)
	}

	return chapters, nil
}

func buildChapterContext(indexedContents []string, currentIndex, contextBefore, contextAfter int) string {
	var contextBuilder strings.Builder

	start := int(math.Max(0, float64(currentIndex-contextBefore)))
	end := int(math.Min(float64(currentIndex+contextAfter), float64(len(indexedContents))))

	contextBuilder.WriteString("Do not include any of the following, this is only to provide context")

	// Add preceding chapters
	contextBuilder.WriteString("\nChapters Before:\n")
	for i := start; i < currentIndex; i++ {
		contextBuilder.WriteString(indexedContents[i])
		contextBuilder.WriteString("\n")
	}

	// Add following chapters
	contextBuilder.WriteString("\nChapters After:\n")
	for i := currentIndex + 1; i < end; i++ {
		contextBuilder.WriteString(indexedContents[i])
	}

	return contextBuilder.String()
}

func isChapter(href string) bool {
	return strings.Contains(strings.ToLower(href), "chapter")
}

func ensure(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0755)
		return err
	}
	return nil
}

func saveEpub(book *epub.Rootfile, dir string) error {
	//meta data
	outputEpub := goepub.NewEpub(book.Title + "_output.epub")
	outputEpub.SetAuthor(book.Title)
	outputEpub.SetDescription(book.Description)
	outputEpub.SetLang(book.Language)

	//chapters
	readDir, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("Failed to load chapters from dir: %v", err)
	}

	for _, file := range readDir {
		if data, err := os.ReadFile(filepath.Join(dir, file.Name())); err != nil {
			return err
		} else {
			_, err := outputEpub.AddSection(string(data), "", file.Name(), "")
			if err != nil {
				return err
			}
		}

	}

	// Save the final EPUB file
	err = outputEpub.Write("output.epub")
	if err != nil {
		return fmt.Errorf("Failed to save output EPUB: %v", err)
	}
	return nil
}
