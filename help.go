/*
 * help.go
 *
 * This file implements the in-game help system that reads from Markdown files.
 * It provides functionality to load, parse, and search help files stored in the
 * "docs" directory. Each help file contains a YAML front matter with title and
 * keywords, followed by Markdown content that is displayed to the player.
 * The file follows a similar pattern to loader.go for loading game data.
 */

package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// HelpFile represents a parsed help file with metadata and content
type HelpFile struct {
	Title    string   `yaml:"title"`
	Keywords []string `yaml:"keywords"`
	Content  string   // The actual help content (not part of YAML)
	Filename string   // The filename for reference
}

// HelpSystem manages the help files and provides search functionality
type HelpSystem struct {
	helpFiles     map[string]*HelpFile // Map of lowercase titles to help files
	keywordIndex  map[string][]string  // Map of keywords to help file titles
	mutex         sync.RWMutex         // For thread-safe access
	docsDirectory string               // Directory where help files are stored
}

// Global help system instance
var helpSystem *HelpSystem

// InitHelpSystem initializes the help system
func InitHelpSystem() {
	helpSystem = NewHelpSystem("docs")
	err := helpSystem.LoadHelpFiles()
	if err != nil {
		log.Printf("Error loading help files: %v", err)
	}
}

// NewHelpSystem creates and initializes a new help system
func NewHelpSystem(docsDir string) *HelpSystem {
	return &HelpSystem{
		helpFiles:     make(map[string]*HelpFile),
		keywordIndex:  make(map[string][]string),
		docsDirectory: docsDir,
	}
}

// LoadHelpFiles loads all Markdown files from the docs directory
func (hs *HelpSystem) LoadHelpFiles() error {
	hs.mutex.Lock()
	defer hs.mutex.Unlock()

	// Clear existing data
	hs.helpFiles = make(map[string]*HelpFile)
	hs.keywordIndex = make(map[string][]string)

	// Create docs directory if it doesn't exist
	if _, err := os.Stat(hs.docsDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(hs.docsDirectory, 0755); err != nil {
			return fmt.Errorf("failed to create docs directory: %w", err)
		}
	}

	// Walk through the docs directory
	err := filepath.WalkDir(hs.docsDirectory, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-markdown files
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		// Parse the help file
		helpFile, err := hs.parseHelpFile(path)
		if err != nil {
			log.Printf("Error parsing help file %s: %v", path, err)
			return nil // Continue with other files
		}

		// Store the help file by its title (lowercase for case-insensitive lookup)
		titleKey := strings.ToLower(helpFile.Title)
		hs.helpFiles[titleKey] = helpFile

		// Index keywords
		for _, keyword := range helpFile.Keywords {
			keyword = strings.ToLower(keyword)
			hs.keywordIndex[keyword] = append(hs.keywordIndex[keyword], helpFile.Title)
		}

		return nil
	})

	// Create a default index help file if it doesn't exist
	if _, exists := hs.helpFiles["index"]; !exists {
		hs.createDefaultIndexFile()
	}

	return err
}

// parseHelpFile reads and parses a Markdown help file
func (hs *HelpSystem) parseHelpFile(filePath string) (*HelpFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Check if the file starts with YAML front matter (---)
	if !scanner.Scan() || scanner.Text() != "---" {
		return nil, fmt.Errorf("help file must start with YAML front matter (---)")
	}

	// Read the YAML front matter
	var yamlContent strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			break
		}
		yamlContent.WriteString(line + "\n")
	}

	// Create a temporary struct to parse the YAML front matter
	type FrontMatter struct {
		Title    string `yaml:"title"`
		Keywords string `yaml:"keywords"`
	}

	var frontMatter FrontMatter
	if err := yaml.Unmarshal([]byte(yamlContent.String()), &frontMatter); err != nil {
		return nil, err
	}

	// Create the help file with parsed data
	helpFile := &HelpFile{
		Title:    frontMatter.Title,
		Filename: filepath.Base(filePath),
	}

	// Split the comma-separated keywords into a slice
	if frontMatter.Keywords != "" {
		keywordsList := strings.Split(frontMatter.Keywords, ",")
		for i, keyword := range keywordsList {
			keywordsList[i] = strings.TrimSpace(keyword)
		}
		helpFile.Keywords = keywordsList
	}

	// Read the rest of the file as content
	var content strings.Builder
	for scanner.Scan() {
		content.WriteString(scanner.Text() + "\n")
	}

	helpFile.Content = content.String()

	return helpFile, nil
}

// createDefaultIndexFile creates a default index help file
func (hs *HelpSystem) createDefaultIndexFile() {
	// Create a list of all available help topics
	var topics strings.Builder
	topics.WriteString("Available Help Topics:\n\n")

	for _, helpFile := range hs.helpFiles {
		topics.WriteString(fmt.Sprintf("- %s\n", helpFile.Title))
	}

	// Create the index help file
	indexFile := &HelpFile{
		Title:    "Index",
		Keywords: []string{"topics", "list", "help"},
		Content:  topics.String(),
		Filename: "index.md",
	}

	hs.helpFiles["index"] = indexFile
}

// GetHelpByTitle looks up a help file by its exact title (case-insensitive)
func (hs *HelpSystem) GetHelpByTitle(title string) *HelpFile {
	hs.mutex.RLock()
	defer hs.mutex.RUnlock()

	return hs.helpFiles[strings.ToLower(title)]
}

// GetHelpByKeyword looks up help files by keyword and returns the first match
func (hs *HelpSystem) GetHelpByKeyword(keyword string) *HelpFile {
	hs.mutex.RLock()
	defer hs.mutex.RUnlock()

	keyword = strings.ToLower(keyword)
	titles, exists := hs.keywordIndex[keyword]
	if !exists || len(titles) == 0 {
		return nil
	}

	// Return the first help file that matches the keyword
	return hs.helpFiles[strings.ToLower(titles[0])]
}

// FormatHelpContent formats the Markdown content for display in-game
// This implementation handles basic Markdown formatting like headers, lists, and code blocks
func (hs *HelpSystem) FormatHelpContent(content string) string {
	lines := strings.Split(content, "\n")
	var formatted strings.Builder

	inCodeBlock := false

	for _, line := range lines {
		// Handle code blocks
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			if inCodeBlock {
				formatted.WriteString("{D}-----------------------------------{x}\n")
			} else {
				formatted.WriteString("{D}-----------------------------------{x}\n")
			}
			continue
		}

		// If we're in a code block, add the line as is with dark gray color
		if inCodeBlock {
			formatted.WriteString("{D}" + line + "{x}\n")
			continue
		}

		// Handle headers
		if strings.HasPrefix(line, "# ") {
			formatted.WriteString("{Y}" + strings.TrimPrefix(line, "# ") + "{x}\n")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			formatted.WriteString("{G}" + strings.TrimPrefix(line, "## ") + "{x}\n")
			continue
		}
		if strings.HasPrefix(line, "### ") {
			formatted.WriteString("{C}" + strings.TrimPrefix(line, "### ") + "{x}\n")
			continue
		}

		// Handle lists
		if strings.HasPrefix(line, "- ") {
			formatted.WriteString("  {C}*{x} " + strings.TrimPrefix(line, "- ") + "\n")
			continue
		}

		// Handle inline code (surrounded by backticks)
		if strings.Contains(line, "`") {
			parts := strings.Split(line, "`")
			for i := 0; i < len(parts); i++ {
				if i%2 == 0 {
					formatted.WriteString(parts[i])
				} else {
					formatted.WriteString("{D}" + parts[i] + "{x}")
				}
			}
			formatted.WriteString("\n")
			continue
		}

		// Regular text
		formatted.WriteString(line + "\n")
	}

	return formatted.String()
}

// RefreshHelpFiles reloads all help files from disk
func (hs *HelpSystem) RefreshHelpFiles() error {
	return hs.LoadHelpFiles()
}

// handleHelp handles the "help" command
func handleHelp(player *Player, args []string) string {
	// If no topic specified, show the index
	topic := "index"
	if len(args) > 0 {
		topic = strings.Join(args, " ")
	}

	// Try to find an exact match by title
	helpFile := helpSystem.GetHelpByTitle(topic)

	// If no exact match, try keywords
	if helpFile == nil {
		helpFile = helpSystem.GetHelpByKeyword(topic)
	}

	// If still no match, show error message
	if helpFile == nil {
		return fmt.Sprintf("No help file found for '%s'. Try 'help index' for a list of topics.", topic)
	}

	// Format the content and return it
	formattedContent := helpSystem.FormatHelpContent(helpFile.Content)
	return fmt.Sprintf("{Y}%s{x}\n\n%s", helpFile.Title, formattedContent)
}
