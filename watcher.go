package watcher

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	// ErrNothingAdded is an error that occurs when a Watcher's Start() method is
	// called and no files or folders have been added to the Watcher's watchlist.
	ErrNothingAdded = errors.New("error: no files added to the watchlist")

	// ErrWatchedFileDeleted is an error that occurs when a file or folder that was
	// being watched has been deleted.
	ErrWatchedFileDeleted = errors.New("error: watched file or folder deleted")
)

// An EventType is a type that is used to describe what type
// of event has occured during the watching process.
type EventType int

const (
	EventFileAdded EventType = 1 << iota
	EventFileDeleted
	EventFileModified
)

// String returns a small string depending on what
// type of event it is.
func (e EventType) String() string {
	switch e {
	case EventFileAdded:
		return "FILE/FOLDER ADDED"
	case EventFileDeleted:
		return "FILE/FOLDER DELETED"
	case EventFileModified:
		return "FILE/FOLDER MODIFIED"
	default:
		return "UNRECOGNIZED EVENT"
	}
}

type Event struct {
	EventType
	os.FileInfo
}

// A Watcher describes a file watcher.
type Watcher struct {
	Names []string
	Event chan Event
	Error chan error

	mu    *sync.Mutex
	Files map[string]os.FileInfo
}

// New returns a new initialized *Watcher.
func New() *Watcher {
	return &Watcher{
		Names: []string{},
		mu:    new(sync.Mutex),
		Files: make(map[string]os.FileInfo),
		Event: make(chan Event),
		Error: make(chan error),
	}
}

// fileInfo is an implementation of os.FileInfo that can be used
// as a mocked os.FileInfo when triggering an event when the specified
// File is nil.
type fileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	sys     interface{}
}

func (fs *fileInfo) IsDir() bool {
	return false
}
func (fs *fileInfo) ModTime() time.Time {
	return fs.modTime
}
func (fs *fileInfo) Mode() os.FileMode {
	return fs.mode
}
func (fs *fileInfo) Name() string {
	return fs.name
}
func (fs *fileInfo) Size() int64 {
	return fs.size
}
func (fs *fileInfo) Sys() interface{} {
	return fs.sys
}

// Trigger is a method that can be used to trigger an event, separate to
// the file watching process.
func (w *Watcher) Trigger(eventType EventType, file os.FileInfo) {
	if file == nil {
		file = &fileInfo{name: "triggered event", modTime: time.Now()}
	}
	w.Event <- Event{eventType, file}
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
		w.Files[fInfo.Name()] = fInfo
		return nil
	}

	// Add all of the os.FileInfo's to w from dir recursively.
	fInfoList, err := ListFiles(name)
	if err != nil {
		return err
	}
	for k, v := range fInfoList {
		w.Files[k] = v
	}
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
		w.mu.Lock()
		delete(w.Files, fInfo.Name())
		w.mu.Unlock()
		return nil
	}

	// Retrieve a list of all of the os.FileInfo's to delete from w.Files.
	fileList, err := ListFiles(name)
	if err != nil {
		return err
	}

	// Remove the appropriate os.FileInfo's from w's os.FileInfo list.
	w.mu.Lock()
	for path := range fileList {
		delete(w.Files, path)
	}
	w.mu.Unlock()
	return nil
}

// Start starts the watching process and checks for changes every `pollInterval`
// amount of milliseconds. If pollInterval is 0, the default is 100ms.
func (w *Watcher) Start(pollInterval int) error {
	if pollInterval == 0 {
		pollInterval = 100
	}

	if len(w.Names) < 1 {
		return ErrNothingAdded
	}

	for {
		fileList := make(map[string]os.FileInfo)
		for _, name := range w.Names {
			// Retrieve the list of os.FileInfo's from w.Name.
			list, err := ListFiles(name)
			if err != nil {
				if os.IsNotExist(err) {
					w.Error <- ErrWatchedFileDeleted
				} else {
					w.Error <- err
				}
			}
			for k, v := range list {
				fileList[k] = v
			}
		}

		if len(fileList) > len(w.Files) {
			// TODO: Return all new files?
			// Check for new files.
			var addedFile os.FileInfo
			for path, fInfo := range fileList {
				if _, found := w.Files[path]; !found {
					addedFile = fInfo
				}
			}
			w.Event <- Event{EventType: EventFileAdded, FileInfo: addedFile}
			w.Files = fileList
		} else if len(fileList) < len(w.Files) {
			// TODO: Return all deleted files?
			//
			// Check for deleted files.
			var deletedFile os.FileInfo
			for path, fInfo := range w.Files {
				if _, found := fileList[path]; !found {
					deletedFile = fInfo
				}
			}
			w.Event <- Event{EventType: EventFileDeleted, FileInfo: deletedFile}
			w.Files = fileList
		}

		// Check for modified files.
		for i, file := range w.Files {
			if fileList[i].ModTime() != file.ModTime() {
				w.Event <- Event{
					EventType: EventFileModified,
					FileInfo:  file,
				}
				w.Files = fileList
				break
			}
		}

		// Sleep for a little bit.
		time.Sleep(time.Millisecond * time.Duration(pollInterval))
	}

	return nil
}

// ListFiles returns a slice of all os.FileInfo's recursively
// contained in a directory. If name is a single file, it returns
// an os.FileInfo slice with a single os.FileInfo.
func ListFiles(name string) (map[string]os.FileInfo, error) {
	var currentDir string

	fileList := make(map[string]os.FileInfo)

	if err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			currentDir = info.Name()
		}

		if err != nil {
			return err
		}

		fileList[filepath.Join(currentDir, info.Name())] = info
		return nil
	}); err != nil {
		return nil, err
	}

	return fileList, nil
}
