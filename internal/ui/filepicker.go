package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
)

// ErrCancelled is returned when the user cancels the file picker.
var ErrCancelled = errors.New("cancelled")

// PickFile presents an interactive keyboard-navigable file browser starting at
// startDir. Only files (not directories) can be selected. Returns the absolute
// path of the chosen file, or ErrCancelled.
func PickFile(startDir string) (string, error) {
	if _, err := os.Stat(startDir); err != nil {
		home, _ := os.UserHomeDir()
		startDir = home
	}

	currentDir, _ := filepath.Abs(startDir)

	for {
		entries, err := os.ReadDir(currentDir)
		if err != nil {
			// Can't enter this dir, back up one level
			parent := filepath.Dir(currentDir)
			if parent == currentDir {
				return "", ErrCancelled
			}
			currentDir = parent
			continue
		}

		// Sort: directories first, then files, alphabetically within each group.
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return entries[i].Name() < entries[j].Name()
		})

		var options []string

		// "Go up" unless already at filesystem root.
		parent := filepath.Dir(currentDir)
		if parent != currentDir {
			options = append(options, ".. (go up)")
		}

		for _, e := range entries {
			if e.IsDir() {
				options = append(options, e.Name()+"/")
			} else {
				options = append(options, e.Name())
			}
		}

		options = append(options, "[Cancel]")

		var choice string
		prompt := &survey.Select{
			Message: fmt.Sprintf("Select SSH key  (%s)", currentDir),
			Options: options,
			Help:    "↑↓ navigate  •  Enter: open dir / select file  •  [Cancel] to go back",
		}

		if err := survey.AskOne(prompt, &choice); err != nil {
			return "", ErrCancelled
		}

		switch {
		case choice == "[Cancel]":
			return "", ErrCancelled

		case choice == ".. (go up)":
			currentDir = filepath.Dir(currentDir)

		case strings.HasSuffix(choice, "/"):
			currentDir = filepath.Join(currentDir, strings.TrimSuffix(choice, "/"))

		default:
			// Regular file selected.
			return filepath.Join(currentDir, choice), nil
		}
	}
}
