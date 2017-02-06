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

	// ErrWatcherClosed is sent through w.Errors when a Watcher is closed.
	ErrWatcherClosed = errors.New("error: watcher was closed")
)

// An Op is a type that is used to describe what type
// of event has occurred during the watching process.
type Op uint32

// Ops
const (
	Create Op = iota
	Write
	Remove
	Rename
	Chmod
)

var ops = map[Op]string{
	Create: "CREATE",
	Write:  "WRITE",
	Remove: "REMOVE",
	Rename: "RENAME",
	Chmod:  "CHMOD",
}

// String prints the string version of the Op consts
func (e Op) String() string {
	if op, found := ops[e]; found {
		return op
	}
	return "UNRECOGNIZED OP"
}

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
// directory and the type of event that's occurred and the full path of the file.
type Event struct {
	Op
	Path string
	os.FileInfo
}

// String returns a string depending on what type of event occurred and the
// file name associated with the event.
func (e Event) String() string {
	if e.FileInfo != nil {
		pathType := "FILE"
		if e.IsDir() {
			pathType = "DIRECTORY"
		}
		return fmt.Sprintf("%s %q %s [%s]", pathType, e.Name(), e.Op, e.Path)
	}
	return "INVALID EVENT"
}

// A Watcher describes a file watcher.
type Watcher struct {
	Event chan Event
	Error chan error

	options []Option

	mu        *sync.Mutex
	wg        *sync.WaitGroup
	running   bool
	files     map[string]os.FileInfo
	ignored   map[string]struct{}
	names     []string
	maxEvents int
}

// New returns a new initialized *Watcher.
// and set a block for gorutines which shouldn't be started before watcher
func New(options ...Option) *Watcher {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	return &Watcher{
		Event:   make(chan Event),
		Error:   make(chan error),
		options: options,
		mu:      new(sync.Mutex),
		wg:      wg,
		files:   make(map[string]os.FileInfo),
		ignored: make(map[string]struct{}),
		names:   []string{},
	}
}

// Ignore adds paths that should be ignored by the watcher.
func (w *Watcher) Ignore(paths ...string) error {
	for _, path := range paths {
		if err := w.Remove(path); err != nil {
			return err
		}
		w.mu.Lock()
		w.ignored[path] = struct{}{}
		w.mu.Unlock()
	}
	return nil
}

// WatchedFiles returns a map of all the files being watched.
func (w *Watcher) WatchedFiles() map[string]os.FileInfo {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.files
}

// SetMaxEvents controls the maximum amount of events that are sent on
// the Event channel per watching cycle. If max events is less than 1, there is
// no limit, which is the default.
func (w *Watcher) SetMaxEvents(amount int) {
	w.mu.Lock()
	w.maxEvents = amount
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
func (w *Watcher) Add(name string) (err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if name == "." || name == ".." {
		name, err = filepath.Abs(name)
		if err != nil {
			return err
		}
	}

	name = filepath.Clean(name)

	// Add the name from w's names list.
	w.names = append(w.names, name)

	// Make sure name exists.
	fInfo, err := os.Stat(name)
	if err != nil {
		return err
	}

	// If watching a single file that isn't in the ignored list,
	// add it and return.
	_, ignored := w.ignored[name]
	if !fInfo.IsDir() && !ignored {
		w.files[fInfo.Name()] = fInfo
		return nil
	}

	// Retrieve a list of all of the os.FileInfo's to add to w.files.
	fInfoList, err := ListFiles(name, w.ignored, w.options...)
	if err != nil {
		return err
	}
	for k, v := range fInfoList {
		w.files[k] = v
	}
	return nil
}

// Remove removes either a single file or recursed directory from
// the Watcher's file list.
func (w *Watcher) Remove(name string) (err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if name == "." || name == ".." {
		name, err = filepath.Abs(name)
		if err != nil {
			return err
		}
	}

	name = filepath.Clean(name)

	// Remove the name from w's names list.
	for i := range w.names {
		if w.names[i] == name {
			w.names = append(w.names[:i], w.names[i+1:]...)
		}
	}

	// If name is a single file, remove it and return.
	info, found := w.files[name]
	if !found {
		return nil // Doesn't exist, just return
	}
	if !info.IsDir() {
		delete(w.files, name)
		return nil
	}

	// If it's a directory, delete all of it's contents from w.files.
	for path := range w.files {
		if strings.HasPrefix(path, name) {
			delete(w.files, path)
		}
	}
	return nil
}

// TriggerEvent is a method that can be used to trigger an event, separate to
// the file watching process.
// This function mandatory wait when the watcher started
func (w *Watcher) TriggerEvent(eventType Op, file os.FileInfo) {
	w.Wait()
	if file == nil {
		file = &fileInfo{name: "triggered event", modTime: time.Now()}
	}
	w.Event <- Event{Op: eventType, Path: "-", FileInfo: file}
}

type renamedFrom struct {
	path string
	os.FileInfo
}

// Close removes all watched files from the watcher and causes Start to
// stop watching and return after the current watching cycle finishes if
// one is currently running.
func (w *Watcher) Close() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = false
	w.Error <- ErrWatcherClosed
	for k, _ := range w.files {
		delete(w.files, k)
	}
	w.names = []string{}
	w.mu.Unlock()
	return nil
}

// Wait blocks until the watcher started.
func (w *Watcher) Wait() {
	w.wg.Wait()
}

// Start starts the watching process and checks for changes every `pollInterval` duration.
// If pollInterval is 0, the default is 100ms.
func (w *Watcher) Start(pollInterval time.Duration) error {
	if pollInterval <= 0 {
		pollInterval = time.Millisecond * 100
	}

	if len(w.names) < 1 {
		return ErrNothingAdded
	}

	w.mu.Lock()
	w.running = true
	w.mu.Unlock()

	w.wg.Done()

	for {
		w.mu.Lock()

		if !w.running {
			w.mu.Unlock()
			return nil
		}

		fileList := make(map[string]os.FileInfo)
		for _, name := range w.names {
			// Retrieve the list of os.FileInfo's from w.Name.
			list, err := ListFiles(name, w.ignored, w.options...)
			if err != nil {
				if os.IsNotExist(err) {
					w.Error <- ErrWatchedFileDeleted
					if err := w.Remove(name); err != nil {
						w.mu.Unlock()
						return err
					}
					continue
				} else {
					w.Error <- err
				}
			}
			for k, v := range list {
				fileList[k] = v
			}
		}

		numEvents := 0

		events := map[Op]map[string]os.FileInfo{
			Create: make(map[string]os.FileInfo),
			Remove: make(map[string]os.FileInfo),
		}

		renamed := make(map[string]renamedFrom)

		// Check for added files.
		for path, file := range fileList {
			if _, found := w.files[path]; !found {
				events[Create][path] = file
			}
		}

		// Check for removed files.
		for path, file := range w.files {
			if _, found := fileList[path]; !found {
				events[Remove][path] = file
			}
		}

		// Check for renamed files.
		for path1, file1 := range events[Create] {
			if w.maxEvents > 0 && numEvents >= w.maxEvents {
				goto SLEEP
			}
			for path2, file2 := range events[Remove] {
				renamed[path2] = renamedFrom{path1, file1}
				if os.SameFile(file1, file2) {
					w.Event <- Event{
						Op:       Rename,
						Path:     path2,
						FileInfo: file2,
					}
					numEvents++

					// Delete path1 from the added files map.
					delete(events[Create], path1)

					// Delete path2 from the deleted files map.
					delete(events[Remove], path2)
				}
			}
		}

		for path, file := range events[Create] {
			if w.maxEvents > 0 && numEvents >= w.maxEvents {
				goto SLEEP
			}
			w.Event <- Event{
				Op:       Create,
				Path:     path,
				FileInfo: file,
			}
			numEvents++
		}

		for path, file := range events[Remove] {
			if w.maxEvents > 0 && numEvents >= w.maxEvents {
				goto SLEEP
			}
			w.Event <- Event{
				Op:       Remove,
				Path:     path,
				FileInfo: file,
			}
			numEvents++
		}

		// Check for modified and chmoded files.
		for path, file := range w.files {
			if w.maxEvents > 0 && numEvents >= w.maxEvents {
				goto SLEEP
			}
			_, addFound := events[Create][path]
			_, removeFound := events[Remove][path]
			renamedFrom, renameFound := renamed[path]
			if !addFound && !removeFound && !renameFound {
				if !file.IsDir() && fileList[path].ModTime() != file.ModTime() {
					w.Event <- Event{
						Op:       Write,
						Path:     path,
						FileInfo: file,
					}
					numEvents++
				}

				if w.maxEvents > 0 && numEvents >= w.maxEvents {
					goto SLEEP
				}
				if fileList[path].Mode() != file.Mode() {
					w.Event <- Event{
						Op:       Chmod,
						Path:     path,
						FileInfo: file,
					}
					numEvents++
				}
			}
			if w.maxEvents > 0 && numEvents >= w.maxEvents {
				goto SLEEP
			}
			if renameFound && renamedFrom.Mode() != file.Mode() {
				w.Event <- Event{
					Op:       Chmod,
					Path:     renamedFrom.path,
					FileInfo: renamedFrom.FileInfo,
				}
				numEvents++
			}
		}

	SLEEP:
		// Update w.files and then sleep for a little bit.
		w.files = fileList
		w.mu.Unlock()
		time.Sleep(pollInterval)
	}
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
func ListFiles(name string, ignoredPaths map[string]struct{}, options ...Option) (map[string]os.FileInfo, error) {
	fileList := make(map[string]os.FileInfo)

	if _, ignored := ignoredPaths[name]; ignored {
		return nil, nil
	}

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

		_, ignored := ignoredPaths[path]
		if ignored || (ignoreDotFiles && strings.HasPrefix(info.Name(), ".")) {
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
