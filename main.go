package main

import (
	"bytes"
	"cmp"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/renameio/v2"
)

var (
	message       = flag.String("m", "", "Title of the snippet. If this is empty then $EDITOR will open to write the snippet, ignoring the -edit flag.")
	edit          = flag.Bool("edit", false, "Open $EDITOR to edit the snippet. Only has effect if -m is specified. If $EDITOR is empty then vim will be used; if vim is not present on the system, an error is returned.")
	timeFormat    = flag.String("time_format", "15:04", "Format of pre-filled timestamp in snippet. Please refer to https://pkg.go.dev/time to read about time formats.")
	includeHeader = flag.Bool("include_header", true, "Include a header containing the current date and timezone as the first line in the snippet file.")
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

// inferLocalTimezone attempts to figure out the IANA name of the local timezone
// (e.g. "Europe/Stockholm" or "America/Los_Angeles"). It's done on best effort
// basis, since macOS doesn't provide any explicit way to query for it.
//
// This function uses the value of the TZ environment variable, if set, as long
// as it is a valid location according to [time.LoadLocation].
func inferLocalTimezone() (string, error) {
	// Let the TZ environment variable take precedence, if it's set and resolves
	// to a valid timezone using [time.LoadLocation].
	if tz := os.Getenv("TZ"); tz != "" {
		if _, err := time.LoadLocation(tz); err == nil { // if NO error
			return tz, nil
		}
	}
	// Best-effort: assume that /etc/localtime is a symlink to a file whose path
	// contains the timezone name in a standardized format. On my macOS system,
	// it looks like this:
	//
	//     $ readlink /etc/localtime
	//     /var/db/timezone/zoneinfo/Europe/London
	//
	// To be a bit more liberal in the paths accepted, look for a "zoneinfo/"
	// substring, and assume everything after it is the timezone name.
	//
	// As a sanity check, try loading the inferred timezone with
	// [time.LoadLocation]. If that doesn't work, return an error.
	const localtime = "/etc/localtime"
	realPath, err := filepath.EvalSymlinks(localtime)
	if err != nil {
		return "", fmt.Errorf("infer local timezone: evaluate %s as a symlink: %w", localtime, err)
	}
	const marker = "zoneinfo/"
	idx := strings.Index(realPath, marker)
	if idx == -1 {
		return "", fmt.Errorf("infer local timezone: infer from %s symlink: real path does not contain %q", localtime, marker)
	}
	inferred := realPath[idx+len(marker):]
	if _, err := time.LoadLocation(inferred); err != nil {
		return "", fmt.Errorf("infer local timezone: infer from %s symlink: inferred timezone %q cannot be loaded with time.LoadLocation: %w", localtime, inferred, err)
	}
	return inferred, nil
}

func run() error {
	openEditor := *edit
	if *message == "" {
		openEditor = true
	}

	if *timeFormat == "" {
		return errors.New("-format is required")
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
	now := time.Now().Local()
	if _, err := tmpFile.WriteString(now.Format(*timeFormat) + " | "); err != nil {
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

	// Assemble the final snippet file and write it out to disk, creating any
	// directories required. To prevent 0-byte or half-written snippet files,
	// write out the result to a temporary file and then atomically move it into
	// place using the github.com/google/renamio package. This might seem
	// excessive for something that's just personal notes stored locally, but
	// for the author of this program these snippets are very valuable, so it's
	// worth being a bit paranoid.
	//
	// The final snippet file should include:
	// * The header, if -include_header=true.
	//     * If the snippet file already includes a header, but
	//       -include_header=false in this invocation, leave the header there
	//       i.e. don't remove it.
	// * Any existing snippet lines.
	// * The new snippet line.

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

	// If the snippet file already exists, read it back in. We might need to add
	// the header, and we need to include any existing snippet lines.
	existing, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		// The file doesn't exist, which is fine, just initialize with empty
		// contents.
		existing = nil
	} else if err != nil {
		// Some other error occurred and we don't know how to handle it.
		return fmt.Errorf("write snippet out to file: read existing snippets: %v", err)
	}
	var assembled bytes.Buffer

	// The only time we need to format the header and write it out is if
	// -include_header=true and the file doesn't already contain a header. In
	// all other cases, we don't need to do anything:
	// * -include_header=true  && contains header        => do nothing
	// * -include_header=false && contains header        => do nothing
	// * -include_header=false && doesn't contain header => do nothing
	// We won't try to parse the header into a date, as that is too fragile.
	// Instead we simply look for whether the file starts with "---", which we
	// use as a proxy for "does the file contain the header".
	if *includeHeader && !bytes.HasPrefix(existing, []byte("---")) {
		timezone, err := inferLocalTimezone()
		if err != nil {
			log.Printf("Failed to infer local timezone: %v", err)
			timezone = "<unknown timezone>"
		}
		headerFormat := "--- Monday Jan _2 2006 in " + timezone + " ---"
		assembled.WriteString(now.Format(headerFormat) + "\n")
	}

	// Include the existing snippets, if any.
	assembled.Write(existing)
	// In case the existing snippets didn't contain a newline, write one out, so
	// that the new snippet is guaranteed to be on a new line. Only do this if
	// there are already any existing snippets -- there should be no _leading_
	// newlines in case the existing snippets are empty (e.g. because this is
	// the first snippet of the day).
	if n := len(existing); n != 0 && existing[n-1] != '\n' {
		assembled.WriteByte('\n')
	}

	// Finally, add the new snippet at the end. Note that we explicitly
	// construct it to hold a newline above, so we don't need to check for/add
	// it here.
	assembled.Write(snippet)

	// Atomically write out the assembled contents to the snippet file.
	if err := renameio.WriteFile(path, assembled.Bytes(), fs.FileMode(0o600)); err != nil {
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
