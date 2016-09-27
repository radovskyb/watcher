package watcher

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

// ErrNothingAdded is an error that occurs when a Watcher's Start() method is
// called and no files or folders have been added to the Watcher's watchlist.
var ErrNothingAdded = errors.New("error: no files added to the watchlist")

// An Event is a type that is used to describe what type
// of event has occured during the watching process.
type Event int

const (
	EventFileAdded Event = 1 << iota
	EventFileDeleted
	EventFileModified
)

// String returns a small string depending on what
// type of event it is.
func (e Event) String() string {
	switch e {
	case EventFileAdded:
		return "FILE/FOLDER ADDED"
	case EventFileDeleted:
		return "FILE/FOLDER DELETED"
	case EventFileModified:
		return "FILE/FOLDER MODIFIED"
	}
	return "UNRECOGNIZED EVENT"
}

// A Watcher describes a file watcher.
type Watcher struct {
	Names []string
	Files []os.FileInfo
	Event chan Event
	Error chan error
}

// New returns a new initialized *Watcher.
func New() *Watcher {
	return &Watcher{
		Names: []string{},
		Files: []os.FileInfo{},
		Event: make(chan Event),
		Error: make(chan error),
	}
}

// Add adds either a single file or recursed directory to
// the Watcher's file list.
func (w *Watcher) Add(name string) error {
	// Add the name from w's Names list.
	w.Names = append(w.Names, name)

	// Make sure name exists.
	fInfo, err := os.Stat(name)
	if err != nil {
		return err
	}

	// If watching a single file, add it and return.
	if !fInfo.IsDir() {
		w.Files = append(w.Files, fInfo)
		return nil
	}

	// Add all of the os.FileInfo's to w from dir recursively.
	fInfoList, err := ListFiles(name)
	if err != nil {
		return err
	}
	w.Files = append(w.Files, fInfoList...)
	return nil
}

// Remove removes either a single file or recursed directory from
// the Watcher's file list.
func (w *Watcher) Remove(name string) error {
	// Remove the name from w's Names list.
	for i := range w.Names {
		if w.Names[i] == name {
			w.Names = append(w.Names[:i], w.Names[i+1:]...)
		}
	}

	// Make sure name exists.
	fInfo, err := os.Stat(name)
	if err != nil {
		return err
	}

	// If name is a single file, remove it and return.
	if !fInfo.IsDir() {
		for i := range w.Files {
			if w.Files[i] == fInfo {
				w.Files = append(w.Files[:i], w.Files[i+1:]...)
			}
		}
		return nil
	}

	// Retrieve a list of all of the os.FileInfo's to delete from w.Files.
	fInfoList, err := ListFiles(name)
	if err != nil {
		return err
	}

	// Remove the appropriate os.FileInfo's from w's os.FileInfo list.
	for i, fInfo := range fInfoList {
		if w.Files[i] == fInfo {
			w.Files = append(w.Files[:i], w.Files[i+1:]...)
		}
	}
	return nil
}

// Start starts the watching process.
func (w *Watcher) Start() error {
	if len(w.Names) < 1 {
		return ErrNothingAdded
	}

	var err error
	var fInfoList []os.FileInfo
	for {
		for _, name := range w.Names {
			// Retrieve the list of os.FileInfo's from w.Name.
			fInfoList, err = ListFiles(name)
			if err != nil {
				w.Error <- err
			}
		}

		// Check for new files.
		if len(fInfoList) > len(w.Files) {
			w.Event <- EventFileAdded
			w.Files = fInfoList
		}

		// Check for deleted files.
		if len(fInfoList) < len(w.Files) {
			w.Event <- EventFileDeleted
			w.Files = fInfoList
		}

		// Check for modified files.
		for i, fInfo := range w.Files {
			if fInfoList[i].ModTime() != fInfo.ModTime() {
				w.Event <- EventFileModified
				w.Files = fInfoList
				break
			}
		}

		// Sleep for a little bit.
		time.Sleep(time.Millisecond * 100)
	}

	return nil
}

// ListFiles returns a slice of all os.FileInfo's recursively
// contained in a directory. If name is a single file, it returns
// an os.FileInfo slice with a single os.FileInfo.
func ListFiles(name string) ([]os.FileInfo, error) {
	fInfoList := []os.FileInfo{}

	if err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		fInfoList = append(fInfoList, info)
		return nil
	}); err != nil {
		return nil, err
	}

	return fInfoList, nil
}
