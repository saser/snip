package main

import (
	"bytes"
	"cmp"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const snippetTimeFormat = time.DateTime + " -0700"

var (
	message = flag.String("m", "", "Title of the snippet. If this is empty then $EDITOR will open to write the snippet, ignoring the -edit flag.")
	edit    = flag.Bool("edit", false, "Open $EDITOR to edit the snippet. Only has effect if -m is specified. If $EDITOR is empty then vim will be used; if vim is not present on the system, an error is returned.")
)

// baseDir returns the base directory for everything related to snip (snippets
// and, potentially in the future, config).
func baseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve snip dir: %v", err)
	}
	return filepath.Join(home, ".snip"), nil
}

// snippetPath is the file path where a snippet timestamped at t should be
// written to.
func snippetPath(t time.Time) (string, error) {
	if t.IsZero() {
		return "", fmt.Errorf("resolve snippet path: timestamp is zero")
	}
	t = t.Local()
	base, err := baseDir()
	if err != nil {
		return "", fmt.Errorf("resolve snippet path: %v", err)
	}
	return filepath.Join(base, t.Format(time.DateOnly)+".txt"), nil
}

func run() error {
	openEditor := *edit
	if *message == "" {
		openEditor = true
	}

	// Create a temporary file to hold the snippet before it's committed to the
	// snipdir.
	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		return fmt.Errorf("create temporary file for editing snippet: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			log.Printf("Deleting temporary file for editing snippet unexpectedly failed: %v", err)
		}
	}()

	// Write the current timestamp as the first part of the snippet.
	// TODO: consider allowing the user to specify this themselves with a `-t
	// 10:30` flag or similar.
	now := time.Now().Local()
	if _, err := tmpFile.WriteString(now.Format(snippetTimeFormat) + " | "); err != nil {
		return fmt.Errorf("write snippet timestamp to temporary file: %v", err)
	}

	// If there is a snippet title prefilled, write it to the temporary file.
	if m := *message; m != "" {
		if _, err := tmpFile.WriteString(m); err != nil {
			return fmt.Errorf("write title from -m to temporary file: %v", err)
		}
	}

	if openEditor {
		editor := cmp.Or(os.Getenv("EDITOR"), "vim")
		cmd := exec.Command(editor, tmpFile.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("open $EDITOR to edit snippet: %v", err)
		}
	}

	// Read the snippet back from the temporary file. After this point, we don't
	// care about the temporary file anymore.
	snippet, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("read temporary file after editing: %v", err)
	}
	snippet = bytes.TrimSpace(snippet)
	if len(snippet) == 0 {
		return fmt.Errorf("snippet is empty")
	}
	// Replace all newlines with spaces, so that each snippet is only on one line.
	snippet = bytes.ReplaceAll(snippet, []byte{'\n'}, []byte{' '})
	// Add a trailing newline.
	snippet = append(snippet, '\n')
	// TODO: add future processing, such as validation, here.

	// Write the snippet out to its file, potentially creating all necessary
	// directories in its path first. If the file already exists, the snippet
	// will be added at the bottom.
	path, err := snippetPath(now)
	if err != nil {
		return fmt.Errorf("write snippet out to file: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), fs.FileMode(0o755)); err != nil {
		return fmt.Errorf("write snippet out to file: ensure directory exists: %v", err)
	}
	snippetFile, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("write snippet out to file: open or create snippet file: %v", err)
	}
	defer func() {
		if err := snippetFile.Close(); err != nil {
			log.Printf("Closing snippet file failed unexpectedly: %v", err)
		}
	}()
	if _, err := snippetFile.Write(snippet); err != nil {
		return fmt.Errorf("write snippet out to file: %v", err)
	}
	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		log.Printf("Fatal error: %v", err)
		os.Exit(1)
	}
}
