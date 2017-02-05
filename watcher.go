// TODO: Fix rename for watching single file.
package watcher

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	// ErrDurationTooShort occurs when calling the poller's Start
	// method with a duration that's less than 1 nanosecond.
	ErrDurationTooShort = errors.New("error: duration is less than 1ns")

	// ErrWatcherRunning occurs when trying to call the poller's
	// Start method and the polling cycle is still already running
	// from previously calling Start and not yet calling Close.
	ErrWatcherRunning = errors.New("error: poller is already running")

	// ErrWatchedFileDeleted is an error that occurs when a file or folder that was
	// being watched has been deleted.
	ErrWatchedFileDeleted = errors.New("error: watched file or folder deleted")
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
	return "???"
}

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
	return "???"
}

type Watcher struct {
	Event  chan Event
	Error  chan error
	Closed chan struct{}
	close  chan struct{}
	pipeline  chan struct{}

	// mu protects the following.
	mu           *sync.Mutex
	wg           *sync.WaitGroup
	running      bool
	names        map[string]bool                   // bool for recursive or not.
	files        map[string]map[string]os.FileInfo // dir to map of files.
	ignored      map[string]struct{}
	ignoreHidden bool // ignore hidden files or not.
	maxEvents    int
}

// New creates a new Watcher
// and set a block for gorutines which shouldn't be started before watcher
func New() *Watcher {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	return &Watcher{
		Event:   make(chan Event),
		Error:   make(chan error),
		Closed:  make(chan struct{}),
		close:   make(chan struct{}),
		mu:      new(sync.Mutex),
		wg:      wg,
		files:   make(map[string]map[string]os.FileInfo),
		ignored: make(map[string]struct{}),
		names:   make(map[string]bool),
	}
}

// SetMaxEvents controls the maximum amount of events that are sent on
// the Event channel per watching cycle. If max events is less than 1, there is
// no limit, which is the default.
func (p *Watcher) SetMaxEvents(amount int) {
	p.mu.Lock()
	p.maxEvents = amount
	p.mu.Unlock()
}

func (p *Watcher) IgnoreHiddenFiles(ignore bool) {
	p.mu.Lock()
	p.ignoreHidden = ignore
	p.mu.Unlock()
}

func (p *Watcher) list(name string) (map[string]map[string]os.FileInfo, error) {
	_, ignored := p.ignored[name]
	if ignored {
		return nil, nil
	}

	fileList := make(map[string]map[string]os.FileInfo)

	dir := filepath.Dir(name)
	if _, found := fileList[dir]; !found {
		fileList[dir] = make(map[string]os.FileInfo)
	}

	stat, err := os.Stat(name)
	if err != nil {
		return nil, err
	}
	if !stat.IsDir() {
		dir := filepath.Dir(name)
		if _, found := fileList[dir]; !found {
			fileList[dir] = make(map[string]os.FileInfo)
		}
		fileList[dir][filepath.Base(name)] = stat
		return fileList, nil
	}

	// It's a directory.
	fInfoList, err := ioutil.ReadDir(name)
	if err != nil {
		return nil, err
	}
	fileList[name] = make(map[string]os.FileInfo)
	for _, fInfo := range fInfoList {
		_, ignored := p.ignored[filepath.Join(name, fInfo.Name())]
		if ignored || (p.ignoreHidden && strings.HasPrefix(fInfo.Name(), ".")) {
			continue
		}
		fileList[name][fInfo.Name()] = fInfo
	}
	return fileList, nil
}

func (p *Watcher) listRecursive(name string) (map[string]map[string]os.FileInfo, error) {
	fileList := make(map[string]map[string]os.FileInfo)
	dir := filepath.Dir(name)
	if _, found := fileList[dir]; !found {
		fileList[dir] = make(map[string]os.FileInfo)
	}
	return fileList, filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// If path is ignored and it's a directory, skip the directory. If it's
		// ignored and it's a single file, skip the file.
		_, ignored := p.ignored[path]
		if ignored || (p.ignoreHidden && strings.HasPrefix(info.Name(), ".")) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			fileList[path] = make(map[string]os.FileInfo)
		}
		// Add the path and it's info to the file list.
		fileList[filepath.Dir(path)][info.Name()] = info
		return nil
	})
}

// Add adds either a single file or directory to the file list.
func (p *Watcher) Add(name string) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	name, err = filepath.Abs(name)
	if err != nil {
		return err
	}

	// Add the name to the names list.
	p.names[name] = false

	// Make sure name exists.
	fInfo, err := os.Stat(name)
	if err != nil {
		return err
	}

	// If hidden files are ignored and name is a hidden file
	// or directory, simply return.
	if p.ignoreHidden && strings.HasPrefix(fInfo.Name(), ".") {
		return nil
	}

	// Add the directory's contents to the files list.
	fileList, err := p.list(name)
	if err != nil {
		return err
	}
	for k, v := range fileList {
		p.files[k] = v
	}

	return nil
}

// Add adds either a single file or directory recursively to the file list.
func (p *Watcher) AddRecursive(name string) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	name, err = filepath.Abs(name)
	if err != nil {
		return err
	}

	// Add the name to the names list.
	p.names[name] = true

	// Make sure name exists.
	fInfo, err := os.Stat(name)
	if err != nil {
		return err
	}

	// If hidden files are ignored and name is a hidden file
	// or directory, simply return.
	if p.ignoreHidden && strings.HasPrefix(fInfo.Name(), ".") {
		return nil
	}

	fileList, err := p.listRecursive(name)
	if err != nil {
		return err
	}
	for k, v := range fileList {
		p.files[k] = v
	}

	return nil
}

// Remove removes either a single file or directory from the file's list.
func (p *Watcher) Remove(name string) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	name, err = filepath.Abs(name)
	if err != nil {
		return err
	}

	// Remove the name from w's names list.
	delete(p.names, name)

	dir, file := filepath.Split(name)

	// If name is a single file, remove it and return.
	info, found := p.files[dir][file]
	if !found {
		return nil // Doesn't exist, just return
	}
	if !info.IsDir() {
		delete(p.files[dir], file)
		return nil
	}

	// If it's a directory, delete all of it's contents from p.files.
	delete(p.files, dir)

	return nil
}

// Remove removes either a single file or a directory recursively from
// the file's list.
func (p *Watcher) RemoveRecursive(name string) (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	name, err = filepath.Abs(name)
	if err != nil {
		return err
	}

	// Remove the name from w's names list.
	delete(p.names, name)

	dir, file := filepath.Split(name)

	// If name is a single file, remove it and return.
	info, found := p.files[dir][file]
	if !found {
		return nil // Doesn't exist, just return
	}
	if !info.IsDir() {
		delete(p.files[dir], file)
		return nil
	}

	// If it's a directory, delete all of it's contents recursively
	// from p.files.
	for d := range p.files {
		if strings.HasPrefix(d, name) {
			delete(p.files, d)
		}
	}
	return nil
}

// Ignore adds paths that should be ignored.
//
// For files that are already added, Ignore removes them.
func (p *Watcher) Ignore(paths ...string) (err error) {
	for _, path := range paths {
		path, err = filepath.Abs(path)
		if err != nil {
			return err
		}
		// Remove any of the paths that were already added.
		if err := p.RemoveRecursive(path); err != nil {
			return err
		}
		p.mu.Lock()
		p.ignored[path] = struct{}{}
		p.mu.Unlock()
	}
	return nil
}

func (p *Watcher) WatchedFiles() map[string]map[string]os.FileInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.files
}

func (p *Watcher) retrieveFileList() map[string]map[string]os.FileInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	fileList := make(map[string]map[string]os.FileInfo)

	for name, recursive := range p.names {
		if recursive {
			list, err := p.listRecursive(name)
			if err == nil {
				for k, v := range list {
					fileList[k] = v
				}
				continue
			}
			if os.IsNotExist(err) {
				p.Error <- ErrWatchedFileDeleted
				p.RemoveRecursive(name)
			} else {
				p.Error <- err
			}
			continue
		}
		list, err := p.list(name)
		if err == nil {
			for k, v := range list {
				fileList[k] = v
			}
			continue
		}
		if os.IsNotExist(err) {
			p.Error <- ErrWatchedFileDeleted
			p.Remove(name)
		} else {
			p.Error <- err
		}
	}

	return fileList
}

// createEvts checks for added files.
func (p *Watcher) createEvts(dir string, files map[string]os.FileInfo,
	cancel chan struct{}) map[string]os.FileInfo {
	creates := make(map[string]os.FileInfo)
	for file, info := range files {
		if _, found := p.files[dir][file]; !found {
			select {
			case <-cancel:
				return nil
			default:
				creates[file] = info
			}
		}
	}
	return creates
}

// removeEvts checks for removed files.
func (p *Watcher) removeEvts(dir string, files map[string]os.FileInfo,
	cancel chan struct{}) map[string]os.FileInfo {
	removes := make(map[string]os.FileInfo)
	for file, info := range p.files[dir] {
		if _, found := files[file]; !found {
			select {
			case <-cancel:
				return nil
			default:
				removes[file] = info
			}
		}
	}
	return removes
}

// writeEvts checks for written files.
func (p *Watcher) writeEvts(dir string, files map[string]os.FileInfo,
	evt chan Event, cancel chan struct{}) {
	for file, info := range p.files[dir] {
		if _, found := files[file]; !found {
			continue
		}
		if files[file].ModTime() != info.ModTime() {
			select {
			case evt <- Event{
				Op:       Write,
				Path:     filepath.Join(dir, file),
				FileInfo: info,
			}:
			case <-cancel:
				return
			}
		}
	}
}

// chmodEvts checks for written files.
func (p *Watcher) chmodEvts(dir string, files map[string]os.FileInfo,
	evt chan Event, cancel chan struct{}) {
	for file, info := range p.files[dir] {
		if _, found := files[file]; !found {
			continue
		}
		if files[file].Mode() != info.Mode() {
			select {
			case evt <- Event{
				Op:       Chmod,
				Path:     filepath.Join(dir, file),
				FileInfo: info,
			}:
			case <-cancel:
				return
			}
		}
	}
}

type renameEvt struct {
	from string
	to   string
	info os.FileInfo
}

func (p *Watcher) renameEvts(dir string, created, removed map[string]os.FileInfo,
	cancel chan struct{}) []*renameEvt {
	renames := []*renameEvt{}
	for cr, add := range created {
		for rm, remove := range removed {
			if add.ModTime() == remove.ModTime() &&
				add.Size() == remove.Size() &&
				add.Mode() == remove.Mode() &&
				add.IsDir() == remove.IsDir() {
				select {
				case <-cancel:
					return nil
				default:
					renames = append(renames, &renameEvt{rm, cr, add})
				}
			}
		}
	}
	return renames
}

func (w *Watcher) Close2() {
	w.close <-1
}

// Wait blocks until the watcher started.
func (w *Watcher) Wait() {
	w.wg.Wait()
}

func (w *Watcher) Start2(pollInterval time.Duration) error {
	if pollInterval < time.Millisecond {
		return ErrDurationTooShort
	}
	if pollInterval <= 0 {
		pollInterval = time.Millisecond * 100
	}
	tick := time.Tick(pollInterval)

	w.wg.Done()

	for {
		select {
		case <-tick:
		case <-w.close:
			return
		}
	}
	return
}

func (p *Watcher) retrieveFileList2() {

	fileList := make(map[string]map[string]os.FileInfo)

	for name, recursive := range p.names {
		if recursive {
			list, err := p.listRecursive(name)
			if err == nil {
				for k, v := range list {
					fileList[k] = v
				}
				continue
			}
			if os.IsNotExist(err) {
				p.Error <- ErrWatchedFileDeleted
				p.RemoveRecursive(name)
			} else {
				p.Error <- err
			}
			continue
		}
		list, err := p.list(name)
		if err == nil {
			for k, v := range list {
				fileList[k] = v
				p.pipeline <- v
			}
			continue
		}
		if os.IsNotExist(err) {
			p.Error <- ErrWatchedFileDeleted
			p.Remove(name)
		} else {
			p.Error <- err
		}
	}

	p.files = fileList
}



// Start begins the polling cycle which repeats every specified
// duration until Close is called.
func (p *Watcher) Start(d time.Duration) error {
	// Return an error if d is less than 1 nanosecond.
	if d < time.Nanosecond {
		return ErrDurationTooShort
	}

	// Make sure the Watcher is not already running.
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return ErrWatcherRunning
	}
	p.running = true
	p.mu.Unlock()

	errc := make(chan error)

	// done lets the inner polling cycle loop know when the
	// current cycle's method has finished executing.
	done := make(chan struct{})

	for {
		evt := make(chan Event)

		// cancel is used to cancel the current cycle's event checking
		// functions, for example, once max events is reached, to avoid
		// memory leaks.
		cancel := make(chan struct{})

		// Retrieve the list of watched files.
		fileList := p.retrieveFileList()

		go func() {
			// Alert the inner loop to continue to the next cycle.
			defer func() { done <- struct{}{} }()

			for dir, files := range fileList {
				p.writeEvts(dir, files, evt, cancel)
				p.chmodEvts(dir, files, evt, cancel)

				creates := p.createEvts(dir, files, cancel)
				removes := p.removeEvts(dir, files, cancel)
				renamed := p.renameEvts(dir, creates, removes, cancel)
				for _, rename := range renamed {
					select {
					case evt <- Event{
						Op: Rename,
						Path: fmt.Sprintf("%s -> %s",
							filepath.Join(dir, rename.from),
							filepath.Join(dir, rename.to)),
						FileInfo: rename.info,
					}:
						delete(removes, rename.from)
						delete(creates, rename.to)
					case <-cancel:
						return
					}
				}
				for cr, info := range creates {
					select {
					case evt <- Event{
						Op:       Create,
						Path:     filepath.Join(dir, cr),
						FileInfo: info,
					}:
					case <-cancel:
						return
					}
				}
				for rm, info := range removes {
					select {
					case evt <- Event{
						Op:       Remove,
						Path:     filepath.Join(dir, rm),
						FileInfo: info,
					}:
					case <-cancel:
						return
					}
				}
			}
		}()

		numEvents := 0

	inner:
		for {
			// Emit any events or errors when they occur.
			select {
			case <-p.close:
				p.Closed <- struct{}{}
				evt = nil
				close(cancel)
				return nil
			case err := <-errc:
				p.Error <- err
			case event := <-evt:
				numEvents++
				if p.maxEvents > 0 && numEvents == p.maxEvents {
					evt = nil
					close(cancel)
				}
				p.Event <- event
			case <-done:
				break inner
			}
		}

		// Sleep for duration d before the next cycle begins.
		p.mu.Lock()
		p.files = fileList
		p.mu.Unlock()

		time.Sleep(d)
	}
}

func (p *Watcher) Close() {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return
	}
	p.running = false
	p.files = make(map[string]map[string]os.FileInfo)
	p.names = make(map[string]bool)
	p.mu.Unlock()
	// Send a close signal to the Start method.
	p.close <- struct{}{}
}
