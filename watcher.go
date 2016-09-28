package watcher

import (
	"errors"
	"os"
	"path/filepath"
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
	File
}

// A Watcher describes a file watcher.
type Watcher struct {
	Names []string
	Files []File
	Event chan Event
	Error chan error
}

// A Watcher describes a file/folder being watched.
type File struct {
	Dir string
	os.FileInfo
}

// New returns a new initialized *Watcher.
func New() *Watcher {
	return &Watcher{
		Names: []string{},
		Files: []File{},
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
func (w *Watcher) Trigger(event EventType, file os.FileInfo) {
	if file == nil {
		file = &fileInfo{name: "triggered event", modTime: time.Now()}
	}
	w.Event <- Event{event, File{FileInfo: file}}
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
		w.Files = append(w.Files, File{Dir: filepath.Dir(fInfo.Name()), FileInfo: fInfo})
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
			if w.Files[i].FileInfo == fInfo &&
				w.Files[i].Dir == filepath.Dir(fInfo.Name()) {
				w.Files = append(w.Files[:i], w.Files[i+1:]...)
			}
		}
		return nil
	}

	// Retrieve a list of all of the os.FileInfo's to delete from w.Files.
	fileList, err := ListFiles(name)
	if err != nil {
		return err
	}

	// Remove the appropriate os.FileInfo's from w's os.FileInfo list.
	for _, file := range fileList {
		for i, wFile := range w.Files {
			if wFile.Dir == file.Dir &&
				wFile.Name() == file.Name() {
				w.Files = append(w.Files[:i], w.Files[i+1:]...)
			}
		}
	}
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
		var fileList []File
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
			fileList = append(fileList, list...)
		}

		if len(fileList) > len(w.Files) {
			// TODO: Return all new files?
			// Check for new files.
			var addedFile File
			for _, file := range fileList {
				if !fileInList(w.Files, file) {
					addedFile = file
				}
			}
			w.Event <- Event{EventType: EventFileAdded, File: addedFile}
			w.Files = fileList
		} else if len(fileList) < len(w.Files) {
			// TODO: Return all deleted files?
			//
			// Check for deleted files.
			var deletedFile File
			for _, file := range w.Files {
				if !fileInList(fileList, file) {
					deletedFile = file
				}
			}
			w.Event <- Event{EventType: EventFileDeleted, File: deletedFile}
			w.Files = fileList
		}

		// Check for modified files.
		for i, file := range w.Files {
			if fileList[i].ModTime() != file.ModTime() {
				w.Event <- Event{
					EventType: EventFileModified,
					File:      file,
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
func ListFiles(name string) ([]File, error) {
	var currentDir string

	fileList := []File{}

	if err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			currentDir = info.Name()
		}

		if err != nil {
			return err
		}

		fileList = append(fileList, File{Dir: currentDir, FileInfo: info})
		return nil
	}); err != nil {
		return nil, err
	}

	return fileList, nil
}

// fileInList checks whether or not a File exists in a []File.
func fileInList(fileList []File, file File) bool {
	for _, f := range fileList {
		if f.Dir == file.Dir && f.Name() == file.Name() {
			return true
		}
	}
	return false
}
