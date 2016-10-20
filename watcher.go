package watcher

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	EventFileRenamed
)

// An Option is a type that is used to set options for a Watcher.
type Option int

const (
	// NonRecursive sets the watcher to not watch directories recursively.
	NonRecursive Option = 1 << iota

	// IgnoreDotFiles sets the watcher to ignore dot files.
	IgnoreDotFiles
)

// An Event desribes an event that is received when files or directory
// changes occur. It includes the os.FileInfo of the changed file or
// directory and the type of event that's occured and the full path of the file.
type Event struct {
	EventType
	Path string
	os.FileInfo
}

// String returns a string depending on what type of event occured and the
// file name associated with the event.
func (e Event) String() string {
	pathType := "FILE"
	if e.IsDir() {
		pathType = "DIRECTORY"
	}

	switch e.EventType {
	case EventFileAdded:
		return fmt.Sprintf("%s %q ADDED [%s]", pathType, e.Name(), e.Path)
	case EventFileDeleted:
		return fmt.Sprintf("%s %q DELETED [%s]", pathType, e.Name(), e.Path)
	case EventFileModified:
		return fmt.Sprintf("%s %q MODIFIED [%s]", pathType, e.Name(), e.Path)
	case EventFileRenamed:
		return fmt.Sprintf("%s %q RENAMED [%s]", pathType, e.Name(), e.Path)
	default:
		return "UNRECOGNIZED EVENT"
	}
}

// A Watcher describes a file watcher.
type Watcher struct {
	Event chan Event
	Error chan error

	options []Option

	maxEventsPerCycle int

	// mu protects Files and Names.
	mu    *sync.Mutex
	Files map[string]os.FileInfo
	Names []string
}

// New returns a new initialized *Watcher.
func New(options ...Option) *Watcher {
	w := &Watcher{
		Event:   make(chan Event),
		Error:   make(chan error),
		options: options,
		mu:      new(sync.Mutex),
		Files:   make(map[string]os.FileInfo),
		Names:   []string{},
	}
	return w
}

// SetMaxEvents controls the maximum amount of events that are sent on
// the Event channel per watching cycle. If max events is less than 1, there is
// no limit, which is the default.
func (w *Watcher) SetMaxEvents(amount int) {
	w.mu.Lock()
	w.maxEventsPerCycle = amount
	w.mu.Unlock()
}

// fileInfo is an implementation of os.FileInfo that can be used
// as a mocked os.FileInfo when triggering an event when the specified
// os.FileInfo is nil.
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

// Add adds either a single file or recursed directory to
// the Watcher's file list.
func (w *Watcher) Add(name string) error {
	// Add the name from w's Names list.
	w.mu.Lock()
	w.Names = append(w.Names, name)
	w.mu.Unlock()

	// Make sure name exists.
	fInfo, err := os.Stat(name)
	if err != nil {
		return err
	}

	// If watching a single file, add it and return.
	if !fInfo.IsDir() {
		w.mu.Lock()
		w.Files[fInfo.Name()] = fInfo
		w.mu.Unlock()
		return nil
	}

	// Retrieve a list of all of the os.FileInfo's to add to w.Files.
	fInfoList, err := ListFiles(name, w.options...)
	if err != nil {
		return err
	}
	w.mu.Lock()
	for k, v := range fInfoList {
		w.Files[k] = v
	}
	w.mu.Unlock()
	return nil
}

// Remove removes either a single file or recursed directory from
// the Watcher's file list.
func (w *Watcher) Remove(name string) error {
	// Remove the name from w's Names list.
	w.mu.Lock()
	for i := range w.Names {
		if w.Names[i] == name {
			w.Names = append(w.Names[:i], w.Names[i+1:]...)
		}
	}
	w.mu.Unlock()

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
	fInfoList, err := ListFiles(name, w.options...)
	if err != nil {
		return err
	}

	// Remove the appropriate os.FileInfo's from w's os.FileInfo list.
	w.mu.Lock()
	for path := range fInfoList {
		delete(w.Files, path)
	}
	w.mu.Unlock()
	return nil
}

// TriggerEvent is a method that can be used to trigger an event, separate to
// the file watching process.
func (w *Watcher) TriggerEvent(eventType EventType, file os.FileInfo) {
	if file == nil {
		file = &fileInfo{name: "triggered event", modTime: time.Now()}
	}
	w.Event <- Event{EventType: eventType, Path: "-", FileInfo: file}
}

// Start starts the watching process and checks for changes every `pollInterval` duration.
// If pollInterval is 0, the default is 100ms.
func (w *Watcher) Start(pollInterval time.Duration) error {
	if pollInterval <= 0 {
		pollInterval = time.Millisecond * 100
	}

	if len(w.Names) < 1 {
		return ErrNothingAdded
	}

	for {
		fileList := make(map[string]os.FileInfo)
		for _, name := range w.Names {
			// Retrieve the list of os.FileInfo's from w.Name.
			list, err := ListFiles(name, w.options...)
			if err != nil {
				if os.IsNotExist(err) {
					w.Error <- ErrWatchedFileDeleted
					// TODO: remove and continue if there is still
					// more than 1 file left after removal.
				} else {
					w.Error <- err
				}
			}
			for k, v := range list {
				fileList[k] = v
			}
		}

		numEvents := 0

		addedAndDeleted := map[EventType]map[string]os.FileInfo{
			EventFileAdded:   make(map[string]os.FileInfo),
			EventFileDeleted: make(map[string]os.FileInfo),
		}

		// Check for added files.
		for path, file := range fileList {
			if _, found := w.Files[path]; !found {
				addedAndDeleted[EventFileAdded][path] = file
				numEvents++
			}
		}

		// Check for deleted files.
		for path, file := range w.Files {
			if _, found := fileList[path]; !found {
				addedAndDeleted[EventFileDeleted][path] = file
			}
		}

		// Check for renamed files.
		for path1, file1 := range addedAndDeleted[EventFileAdded] {
			if w.maxEventsPerCycle > 0 && numEvents > w.maxEventsPerCycle {
				goto SLEEP
			}
			for path2, file2 := range addedAndDeleted[EventFileDeleted] {
				if path1 != path2 && filepath.Dir(path1) == filepath.Dir(path2) &&
					file1.IsDir() == file2.IsDir() &&
					file1.ModTime() == file2.ModTime() && // TODO: Check this <--
					file1.Mode() == file2.Mode() &&
					file1.Size() == file2.Size() {
					w.Event <- Event{
						EventType: EventFileRenamed,
						Path:      path2,
						FileInfo:  file2,
					}
					numEvents++

					// TODO: check which ones actually need deleting.. lazy delete atm.
					delete(addedAndDeleted[EventFileAdded], path1)
					delete(addedAndDeleted[EventFileAdded], path2)
					delete(addedAndDeleted[EventFileDeleted], path1)
					delete(addedAndDeleted[EventFileDeleted], path2)
					delete(w.Files, path2)
					delete(w.Files, path1)
				}
			}
		}

		for path, file := range addedAndDeleted[EventFileAdded] {
			if w.maxEventsPerCycle > 0 && numEvents > w.maxEventsPerCycle {
				goto SLEEP
			}
			w.Event <- Event{
				EventType: EventFileAdded,
				Path:      path,
				FileInfo:  file,
			}
			numEvents++
		}

		for path, file := range addedAndDeleted[EventFileDeleted] {
			if w.maxEventsPerCycle > 0 && numEvents > w.maxEventsPerCycle {
				goto SLEEP
			}
			w.Event <- Event{
				EventType: EventFileDeleted,
				Path:      path,
				FileInfo:  file,
			}
			numEvents++
		}

		// Check for modified files.
		for path, file := range w.Files {
			if w.maxEventsPerCycle > 0 && numEvents > w.maxEventsPerCycle {
				goto SLEEP
			}
			_, addedFound := addedAndDeleted[EventFileAdded][path]
			_, deletedFound := addedAndDeleted[EventFileDeleted][path]
			if !addedFound && !deletedFound {
				if fileList[path].ModTime() != file.ModTime() {
					w.Event <- Event{
						EventType: EventFileModified,
						Path:      path,
						FileInfo:  file,
					}
					numEvents++
				}
			}
		}

	SLEEP:
		// Update w.Files and then sleep for a little bit.
		w.Files = fileList
		time.Sleep(pollInterval)
	}

	return nil
}

// hasOption returns true or false based on whether or not
// an Option exists in an Option slice.
func hasOption(options []Option, option Option) bool {
	for _, o := range options {
		if option&o != 0 {
			return true
		}
	}
	return false
}

// ListFiles returns a map of all os.FileInfo's recursively
// contained in a directory. If name is a single file, it returns
// an os.FileInfo map containing a single os.FileInfo.
func ListFiles(name string, options ...Option) (map[string]os.FileInfo, error) {
	fileList := make(map[string]os.FileInfo)

	name = filepath.Clean(name)

	nonRecursive := hasOption(options, NonRecursive)
	ignoreDotFiles := hasOption(options, IgnoreDotFiles)

	if nonRecursive {
		f, err := os.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		info, err := os.Stat(name)
		if err != nil {
			return nil, err
		}
		// Add the name to fileList.
		if !info.IsDir() && ignoreDotFiles && strings.HasPrefix(name, ".") {
			return fileList, nil
		}
		fileList[name] = info
		if !info.IsDir() {
			return fileList, nil
		}
		// It's a directory, read it's contents.
		fInfoList, err := f.Readdir(-1)
		if err != nil {
			return nil, err
		}
		// Add all of the FileInfo's returned from f.ReadDir to fileList.
		for _, fInfo := range fInfoList {
			if ignoreDotFiles && strings.HasPrefix(fInfo.Name(), ".") {
				continue
			}
			fileList[filepath.Join(name, fInfo.Name())] = fInfo
		}
		return fileList, nil
	}

	if err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if ignoreDotFiles && strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		fileList[path] = info

		return nil
	}); err != nil {
		return nil, err
	}

	return fileList, nil
}
