package watcher

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// setup creates all required files and folders for
// the tests and returns a function that is used as
// a teardown function when the tests are done.
func setup(t testing.TB) (string, func()) {
	testDir, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(filepath.Join(testDir, "file.txt"),
		[]byte{}, 0755)
	if err != nil {
		t.Fatal(err)
	}

	files := []string{"file_1.txt", "file_2.txt", "file_3.txt"}

	for _, f := range files {
		filePath := filepath.Join(testDir, f)
		if err := ioutil.WriteFile(filePath, []byte{}, 0755); err != nil {
			t.Fatal(err)
		}
	}

	err = ioutil.WriteFile(filepath.Join(testDir, ".dotfile"),
		[]byte{}, 0755)
	if err != nil {
		t.Fatal(err)
	}

	testDirTwo := filepath.Join(testDir, "testDirTwo")
	err = os.Mkdir(testDirTwo, 0755)
	if err != nil {
		t.Fatal(err)
	}

	err = ioutil.WriteFile(filepath.Join(testDirTwo, "file_recursive.txt"),
		[]byte{}, 0755)
	if err != nil {
		t.Fatal(err)
	}

	abs, err := filepath.Abs(testDir)
	if err != nil {
		os.RemoveAll(testDir)
		t.Fatal(err)
	}
	return abs, func() {
		if os.RemoveAll(testDir); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEventString(t *testing.T) {
	e := &Event{Op: Create, Path: "/fake/path"}

	testCases := []struct {
		info     os.FileInfo
		expected string
	}{
		{nil, "???"},
		{
			&fileInfo{name: "f1", dir: true},
			"DIRECTORY \"f1\" CREATE [/fake/path]",
		},
		{
			&fileInfo{name: "f2", dir: false},
			"FILE \"f2\" CREATE [/fake/path]",
		},
	}

	for _, tc := range testCases {
		e.FileInfo = tc.info
		if e.String() != tc.expected {
			t.Errorf("expected e.String() to be %s, got %s", tc.expected, e.String())
		}
	}
}

func TestFileInfo(t *testing.T) {
	modTime := time.Now()

	fInfo := &fileInfo{
		name:    "finfo",
		size:    1,
		mode:    os.ModeDir,
		modTime: modTime,
		sys:     nil,
		dir:     true,
	}

	// Test file info methods.
	if fInfo.Name() != "finfo" {
		t.Fatalf("expected fInfo.Name() to be 'finfo', got %s", fInfo.Name())
	}
	if fInfo.IsDir() != true {
		t.Fatalf("expected fInfo.IsDir() to be true, got %t", fInfo.IsDir())
	}
	if fInfo.Size() != 1 {
		t.Fatalf("expected fInfo.Size() to be 1, got %d", fInfo.Size())
	}
	if fInfo.Sys() != nil {
		t.Fatalf("expected fInfo.Sys() to be nil, got %v", fInfo.Sys())
	}
	if fInfo.ModTime() != modTime {
		t.Fatalf("expected fInfo.ModTime() to be %v, got %v", modTime, fInfo.ModTime())
	}
	if fInfo.Mode() != os.ModeDir {
		t.Fatalf("expected fInfo.Mode() to be os.ModeDir, got %#v", fInfo.Mode())
	}

	w := New()

	w.wg.Done() // Set the waitgroup to done.

	go func() {
		// Trigger an event with the file info.
		w.TriggerEvent(Create, fInfo)
	}()

	e := <-w.Event

	if e.FileInfo != fInfo {
		t.Fatal("expected e.FileInfo to be equal to fInfo")
	}
}

func TestWatcherAdd(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	// Try to add a non-existing path.
	err := w.Add("-")
	if err == nil {
		t.Error("expected error to not be nil")
	}

	if err := w.Add(testDir); err != nil {
		t.Fatal(err)
	}

	if len(w.files) != 7 {
		t.Errorf("expected len(w.files) to be 7, got %d", len(w.files))
	}

	// Make sure w.names contains testDir
	if _, found := w.names[testDir]; !found {
		t.Errorf("expected w.names to contain testDir")
	}

	if _, found := w.files[testDir]; !found {
		t.Errorf("expected to find %s", testDir)
	}

	if w.files[testDir].Name() != filepath.Base(testDir) {
		t.Errorf("expected w.files[%q].Name() to be %s, got %s",
			testDir, testDir, w.files[testDir].Name())
	}

	dotFile := filepath.Join(testDir, ".dotfile")
	if _, found := w.files[dotFile]; !found {
		t.Errorf("expected to find %s", dotFile)
	}

	if w.files[dotFile].Name() != ".dotfile" {
		t.Errorf("expected w.files[%q].Name() to be .dotfile, got %s",
			dotFile, w.files[dotFile].Name())
	}

	fileRecursive := filepath.Join(testDir, "testDirTwo", "file_recursive.txt")
	if _, found := w.files[fileRecursive]; found {
		t.Errorf("expected to not find %s", fileRecursive)
	}

	fileTxt := filepath.Join(testDir, "file.txt")
	if _, found := w.files[fileTxt]; !found {
		t.Errorf("expected to find %s", fileTxt)
	}

	if w.files[fileTxt].Name() != "file.txt" {
		t.Errorf("expected w.files[%q].Name() to be file.txt, got %s",
			fileTxt, w.files[fileTxt].Name())
	}

	dirTwo := filepath.Join(testDir, "testDirTwo")
	if _, found := w.files[dirTwo]; !found {
		t.Errorf("expected to find %s directory", dirTwo)
	}

	if w.files[dirTwo].Name() != "testDirTwo" {
		t.Errorf("expected w.files[%q].Name() to be testDirTwo, got %s",
			dirTwo, w.files[dirTwo].Name())
	}
}

func TestIgnore(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	err := w.Add(testDir)
	if err != nil {
		t.Errorf("expected error to be nil, got %s", err)
	}
	if len(w.files) != 7 {
		t.Errorf("expected len(w.files) to be 7, got %d", len(w.files))
	}

	err = w.Ignore(testDir)
	if err != nil {
		t.Errorf("expected error to be nil, got %s", err)
	}
	if len(w.files) != 0 {
		t.Errorf("expected len(w.files) to be 0, got %d", len(w.files))
	}

	// Now try to add the ignored directory.
	err = w.Add(testDir)
	if err != nil {
		t.Errorf("expected error to be nil, got %s", err)
	}
	if len(w.files) != 0 {
		t.Errorf("expected len(w.files) to be 0, got %d", len(w.files))
	}
}

func TestRemove(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	err := w.Add(testDir)
	if err != nil {
		t.Errorf("expected error to be nil, got %s", err)
	}
	if len(w.files) != 7 {
		t.Errorf("expected len(w.files) to be 7, got %d", len(w.files))
	}

	err = w.Remove(testDir)
	if err != nil {
		t.Errorf("expected error to be nil, got %s", err)
	}
	if len(w.files) != 0 {
		t.Errorf("expected len(w.files) to be 0, got %d", len(w.files))
	}

	// TODO: Test remove single file.
}

// TODO: Test remove recursive function.

func TestIgnoreHiddenFilesRecursive(t *testing.T) {
	// TODO: Write tests for ignore hidden on windows.
	if runtime.GOOS == "windows" {
		return
	}

	testDir, teardown := setup(t)
	defer teardown()

	w := New()
	w.IgnoreHiddenFiles(true)

	if err := w.AddRecursive(testDir); err != nil {
		t.Fatal(err)
	}

	if len(w.files) != 7 {
		t.Errorf("expected len(w.files) to be 7, got %d", len(w.files))
	}

	// Make sure w.names contains testDir
	if _, found := w.names[testDir]; !found {
		t.Errorf("expected w.names to contain testDir")
	}

	if _, found := w.files[testDir]; !found {
		t.Errorf("expected to find %s", testDir)
	}

	if w.files[testDir].Name() != filepath.Base(testDir) {
		t.Errorf("expected w.files[%q].Name() to be %s, got %s",
			testDir, filepath.Base(testDir), w.files[testDir].Name())
	}

	fileRecursive := filepath.Join(testDir, "testDirTwo", "file_recursive.txt")
	if _, found := w.files[fileRecursive]; !found {
		t.Errorf("expected to find %s", fileRecursive)
	}

	if _, found := w.files[filepath.Join(testDir, ".dotfile")]; found {
		t.Error("expected to not find .dotfile")
	}

	fileTxt := filepath.Join(testDir, "file.txt")
	if _, found := w.files[fileTxt]; !found {
		t.Errorf("expected to find %s", fileTxt)
	}

	if w.files[fileTxt].Name() != "file.txt" {
		t.Errorf("expected w.files[%q].Name() to be file.txt, got %s",
			fileTxt, w.files[fileTxt].Name())
	}

	dirTwo := filepath.Join(testDir, "testDirTwo")
	if _, found := w.files[dirTwo]; !found {
		t.Errorf("expected to find %s directory", dirTwo)
	}

	if w.files[dirTwo].Name() != "testDirTwo" {
		t.Errorf("expected w.files[%q].Name() to be testDirTwo, got %s",
			dirTwo, w.files[dirTwo].Name())
	}
}

func TestIgnoreHiddenFiles(t *testing.T) {
	// TODO: Write tests for ignore hidden on windows.
	if runtime.GOOS == "windows" {
		return
	}

	testDir, teardown := setup(t)
	defer teardown()

	w := New()
	w.IgnoreHiddenFiles(true)

	if err := w.Add(testDir); err != nil {
		t.Fatal(err)
	}

	if len(w.files) != 6 {
		t.Errorf("expected len(w.files) to be 6, got %d", len(w.files))
	}

	// Make sure w.names contains testDir
	if _, found := w.names[testDir]; !found {
		t.Errorf("expected w.names to contain testDir")
	}

	if _, found := w.files[testDir]; !found {
		t.Errorf("expected to find %s", testDir)
	}

	if w.files[testDir].Name() != filepath.Base(testDir) {
		t.Errorf("expected w.files[%q].Name() to be %s, got %s",
			testDir, filepath.Base(testDir), w.files[testDir].Name())
	}

	if _, found := w.files[filepath.Join(testDir, ".dotfile")]; found {
		t.Error("expected to not find .dotfile")
	}

	fileRecursive := filepath.Join(testDir, "testDirTwo", "file_recursive.txt")
	if _, found := w.files[fileRecursive]; found {
		t.Errorf("expected to not find %s", fileRecursive)
	}

	fileTxt := filepath.Join(testDir, "file.txt")
	if _, found := w.files[fileTxt]; !found {
		t.Errorf("expected to find %s", fileTxt)
	}

	if w.files[fileTxt].Name() != "file.txt" {
		t.Errorf("expected w.files[%q].Name() to be file.txt, got %s",
			fileTxt, w.files[fileTxt].Name())
	}

	dirTwo := filepath.Join(testDir, "testDirTwo")
	if _, found := w.files[dirTwo]; !found {
		t.Errorf("expected to find %s directory", dirTwo)
	}

	if w.files[dirTwo].Name() != "testDirTwo" {
		t.Errorf("expected w.files[%q].Name() to be testDirTwo, got %s",
			dirTwo, w.files[dirTwo].Name())
	}
}

func TestWatcherAddRecursive(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	if err := w.AddRecursive(testDir); err != nil {
		t.Fatal(err)
	}

	// Make sure len(w.files) is 8.
	if len(w.files) != 8 {
		t.Errorf("expected 8 files, found %d", len(w.files))
	}

	// Make sure w.names contains testDir
	if _, found := w.names[testDir]; !found {
		t.Errorf("expected w.names to contain testDir")
	}

	dirTwo := filepath.Join(testDir, "testDirTwo")
	if _, found := w.files[dirTwo]; !found {
		t.Errorf("expected to find %s directory", dirTwo)
	}

	if w.files[dirTwo].Name() != "testDirTwo" {
		t.Errorf("expected w.files[%q].Name() to be testDirTwo, got %s",
			"testDirTwo", w.files[dirTwo].Name())
	}

	fileRecursive := filepath.Join(dirTwo, "file_recursive.txt")
	if _, found := w.files[fileRecursive]; !found {
		t.Errorf("expected to find %s directory", fileRecursive)
	}

	if w.files[fileRecursive].Name() != "file_recursive.txt" {
		t.Errorf("expected w.files[%q].Name() to be file_recursive.txt, got %s",
			fileRecursive, w.files[fileRecursive].Name())
	}
}

func TestWatcherAddNotFound(t *testing.T) {
	w := New()

	// Make sure there is an error when adding a
	// non-existent file/folder.
	if err := w.AddRecursive("random_filename.txt"); err == nil {
		t.Error("expected a file not found error")
	}
}

func TestWatcherRemoveRecursive(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	// Add the testDir to the watchlist.
	if err := w.AddRecursive(testDir); err != nil {
		t.Fatal(err)
	}

	// Make sure len(w.files) is 8.
	if len(w.files) != 8 {
		t.Errorf("expected 8 files, found %d", len(w.files))
	}

	// Now remove the folder from the watchlist.
	if err := w.RemoveRecursive(testDir); err != nil {
		t.Error(err)
	}

	// Now check that there is nothing being watched.
	if len(w.files) != 0 {
		t.Errorf("expected len(w.files) to be 0, got %d", len(w.files))
	}

	// Make sure len(w.names) is now 0.
	if len(w.names) != 0 {
		t.Errorf("expected len(w.names) to be empty, len(w.names): %d", len(w.names))
	}
}

func TestListFiles(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()
	w.AddRecursive(testDir)

	fileList := w.retrieveFileList()
	if fileList == nil {
		t.Error("expected file list to not be empty")
	}

	// Make sure fInfoTest contains the correct os.FileInfo names.
	fname := filepath.Join(testDir, "file.txt")
	if fileList[fname].Name() != "file.txt" {
		t.Errorf("expected fileList[%s].Name() to be file.txt, got %s",
			fname, fileList[fname].Name())
	}

	// Try to call list on a file that's not a directory.
	fileList, err := w.list(fname)
	if err != nil {
		t.Error("expected err to be nil")
	}
	if len(fileList) != 1 {
		t.Errorf("expected len of file list to be 1, got %d", len(fileList))
	}
}

func TestTriggerEvent(t *testing.T) {
	w := New()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		select {
		case event := <-w.Event:
			if event.Name() != "triggered event" {
				t.Errorf("expected event file name to be triggered event, got %s",
					event.Name())
			}
		case <-time.After(time.Millisecond * 250):
			t.Fatal("received no event from Event channel")
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Fatal(err)
		}
	}()

	w.TriggerEvent(Create, nil)

	wg.Wait()
}

func TestScanNow(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()
	w.FilterOps(Create)

	// Add the testDir to the watchlist.
	if err := w.AddRecursive(testDir); err != nil {
		t.Fatal(err)
	}

	// should not be able to ScanNow() before the watcher is started
	if err := w.ScanNow(); err != ErrWatcherNotRunning {
		t.Fatal("expected an ErrWatcherNotRunning error, but didn't get one")
	}

	testFilePath := filepath.Join(testDir, "test_file1.txt")
	done := make(chan struct{})
	go func() {
		evt := <-w.Event
		if evt.Op == Create && evt.Path == testFilePath {
			close(done)
		} else {
			t.Fatal("unexpected event")
		}
	}()

	// Start scanning with a very long poll duration
	go func() {
		if err := w.Start(time.Hour); err != nil {
			t.Fatal(err)
		}
	}()

	w.Wait()
	defer w.Close()

	// perform initial scan, which should yield no changes
	// this ensures the initial scan has happened, and means the watcher is now waiting 1hr before scanning again
	if err := w.ScanNow(); err != nil {
		t.Error(err)
	}

	// wait for a short period just to ensure no unexpected events arrive
	select {
	case <-time.After(time.Millisecond * 100):
	case <-done:
		t.Fatal("should not have received an event as no changes have occurred since ScanNow() completed")
	}

	// create the test file, we will not receive events due to the 1hr poll duration
	if err := ioutil.WriteFile(testFilePath, []byte{}, 0755); err != nil {
		t.Error(err)
	}

	// wait for a short period just to ensure no unexpected events arrive now we've changed a file
	select {
	case <-time.After(time.Millisecond * 100):
	case <-done:
		t.Fatal("should not have received an event as a poll duration of 1 hour is used")
	}

	// issue a scan now, and we will receive the events while ScanNow() is running.
	if err := w.ScanNow(); err != nil {
		t.Error(err)
	}

	// all events should have been received *whilst* ScanNow() was running, so the done channel should already be
	// closed
	select {
	case <-done:
	default:
		t.Fatal("events from ScanNow() should have been received before ScanNow() returned")
	}

	w.Close()

	// issue a scan now after closing, should error
	if err := w.ScanNow(); err != ErrWatcherNotRunning {
		t.Fatal("expected an ErrWatcherNotRunning error, but didn't get one")
	}
}

func TestSizeOnlyChange(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()
	w.FilterOps(Write)

	// Add the testDir to the watchlist.
	testFilePath := filepath.Join(testDir, "file.txt")
	if err := w.Add(testFilePath); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	go func() {
		evt := <-w.Event
		if evt.Op == Write && evt.Path == testFilePath {
			close(done)
		} else {
			t.Fatal("unexpected event")
		}
	}()

	// Start scanning with a very long poll duration
	go func() {
		if err := w.Start(time.Hour); err != nil {
			t.Fatal(err)
		}
	}()

	w.Wait()
	defer w.Close()

	// perform initial scan, which should yield no changes
	// this ensures the initial scan has happened, and means the watcher is now waiting 1hr before scanning again
	if err := w.ScanNow(); err != nil {
		t.Error(err)
	}

	// modify the test file, we will not receive events due to the 1hr poll duration
	// when modifying, we ensure the mod time does not change. this tests the situation where a file system has to
	// be able to detect multiple file changes within its mod time resolution, which on some systems can be 1 or 2
	// seconds. the watcher should detect the change because the size of the file has changed.
	stat, err := os.Stat(testFilePath)
	if err != nil {
		t.Error(err)
	}
	if err = ioutil.WriteFile(testFilePath, []byte("bigger than before"), 0755); err != nil {
		t.Error(err)
	}
	if err = os.Chtimes(testFilePath, stat.ModTime(), stat.ModTime()); err != nil {
		t.Error(err)
	}

	// issue a scan now, and we will receive the events while ScanNow() is running.
	if err := w.ScanNow(); err != nil {
		t.Error(err)
	}

	// all events should have been received *whilst* ScanNow() was running, but our handler may still be processing
	// the event, so we'll wait for a little while
	// closed
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Did not detect a size only change (no mod time change)")
	}

	w.Close()
}

func TestEventAddFile(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()
	w.FilterOps(Create)

	// Add the testDir to the watchlist.
	if err := w.AddRecursive(testDir); err != nil {
		t.Fatal(err)
	}

	files := map[string]bool{
		"newfile_1.txt": false,
		"newfile_2.txt": false,
		"newfile_3.txt": false,
	}

	for f := range files {
		filePath := filepath.Join(testDir, f)
		if err := ioutil.WriteFile(filePath, []byte{}, 0755); err != nil {
			t.Error(err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		events := 0
		for {
			select {
			case event := <-w.Event:
				if event.Op != Create {
					t.Errorf("expected event to be Create, got %s", event.Op)
				}

				files[event.Name()] = true
				events++

				// Check Path and OldPath content
				newFile := filepath.Join(testDir, event.Name())
				if event.Path != newFile {
					t.Errorf("Event.Path should be %s but got %s", newFile, event.Path)
				}
				if event.OldPath != "" {
					t.Errorf("Event.OldPath should be empty on create, but got %s", event.OldPath)
				}

				if events == len(files) {
					return
				}
			case <-time.After(time.Millisecond * 250):
				for f, e := range files {
					if !e {
						t.Errorf("received no event for file %s", f)
					}
				}
				return
			}
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Fatal(err)
		}
	}()

	wg.Wait()
}

// TODO: TestIgnoreFiles
func TestIgnoreFiles(t *testing.T) {}

func TestEventDeleteFile(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()
	w.FilterOps(Remove)

	// Add the testDir to the watchlist.
	if err := w.AddRecursive(testDir); err != nil {
		t.Fatal(err)
	}

	files := map[string]bool{
		"file_1.txt": false,
		"file_2.txt": false,
		"file_3.txt": false,
	}

	for f := range files {
		filePath := filepath.Join(testDir, f)
		if err := os.Remove(filePath); err != nil {
			t.Error(err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		events := 0
		for {
			select {
			case event := <-w.Event:
				if event.Op != Remove {
					t.Errorf("expected event to be Remove, got %s", event.Op)
				}

				files[event.Name()] = true
				events++

				if events == len(files) {
					return
				}
			case <-time.After(time.Millisecond * 250):
				for f, e := range files {
					if !e {
						t.Errorf("received no event for file %s", f)
					}
				}
				return
			}
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Fatal(err)
		}
	}()

	wg.Wait()
}

func TestEventRenameFile(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	srcFilename := "file.txt"
	dstFilename := "file1.txt"

	w := New()
	w.FilterOps(Rename)

	// Add the testDir to the watchlist.
	if err := w.AddRecursive(testDir); err != nil {
		t.Fatal(err)
	}

	// Rename a file.
	if err := os.Rename(
		filepath.Join(testDir, srcFilename),
		filepath.Join(testDir, dstFilename),
	); err != nil {
		t.Error(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		select {
		case event := <-w.Event:
			if event.Op != Rename {
				t.Errorf("expected event to be Rename, got %s", event.Op)
			}

			// Check Path and OldPath content
			oldFile := filepath.Join(testDir, srcFilename)
			newFile := filepath.Join(testDir, dstFilename)
			if event.Path != newFile {
				t.Errorf("Event.Path should be %s but got %s", newFile, event.Path)
			}
			if event.OldPath != oldFile {
				t.Errorf("Event.OldPath should %s but got %s", oldFile, event.OldPath)
			}

		case <-time.After(time.Millisecond * 250):
			t.Fatal("received no rename event")
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Fatal(err)
		}
	}()

	wg.Wait()
}

func TestEventChmodFile(t *testing.T) {
	// Chmod is not supported under windows.
	if runtime.GOOS == "windows" {
		return
	}

	testDir, teardown := setup(t)
	defer teardown()

	w := New()
	w.FilterOps(Chmod)

	// Add the testDir to the watchlist.
	if err := w.Add(testDir); err != nil {
		t.Fatal(err)
	}

	files := map[string]bool{
		"file_1.txt": false,
		"file_2.txt": false,
		"file_3.txt": false,
	}

	for f := range files {
		filePath := filepath.Join(testDir, f)
		if err := os.Chmod(filePath, os.ModePerm); err != nil {
			t.Error(err)
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		events := 0
		for {
			select {
			case event := <-w.Event:
				if event.Op != Chmod {
					t.Errorf("expected event to be Remove, got %s", event.Op)
				}

				files[event.Name()] = true
				events++

				if events == len(files) {
					return
				}
			case <-time.After(time.Millisecond * 250):
				for f, e := range files {
					if !e {
						t.Errorf("received no event for file %s", f)
					}
				}
				return
			}
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Fatal(err)
		}
	}()

	wg.Wait()
}

func TestWatcherStartWithInvalidDuration(t *testing.T) {
	w := New()

	err := w.Start(0)
	if err != ErrDurationTooShort {
		t.Fatalf("expected ErrDurationTooShort error, got %s", err.Error())
	}
}

func TestWatcherStartWhenAlreadyRunning(t *testing.T) {
	w := New()

	go func() {
		err := w.Start(time.Millisecond * 100)
		if err != nil {
			t.Fatal(err)
		}
	}()
	w.Wait()

	err := w.Start(time.Millisecond * 100)
	if err != ErrWatcherRunning {
		t.Fatalf("expected ErrWatcherRunning error, got %s", err.Error())
	}
}

func BenchmarkEventRenameFile(b *testing.B) {
	testDir, teardown := setup(b)
	defer teardown()

	w := New()
	w.FilterOps(Rename)

	// Add the testDir to the watchlist.
	if err := w.AddRecursive(testDir); err != nil {
		b.Fatal(err)
	}

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond); err != nil {
			b.Fatal(err)
		}
	}()

	var filenameFrom = filepath.Join(testDir, "file.txt")
	var filenameTo = filepath.Join(testDir, "file1.txt")

	for i := 0; i < b.N; i++ {
		// Rename a file.
		if err := os.Rename(
			filenameFrom,
			filenameTo,
		); err != nil {
			b.Error(err)
		}

		select {
		case event := <-w.Event:
			if event.Op != Rename {
				b.Errorf("expected event to be Rename, got %s", event.Op)
			}
		case <-time.After(time.Millisecond * 250):
			b.Fatal("received no rename event")
		}

		filenameFrom, filenameTo = filenameTo, filenameFrom
	}
}

func BenchmarkListFiles(b *testing.B) {
	testDir, teardown := setup(b)
	defer teardown()

	w := New()
	err := w.AddRecursive(testDir)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		fileList := w.retrieveFileList()
		if fileList == nil {
			b.Fatal("expected file list to not be empty")
		}
	}
}

func TestClose(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	err := w.Add(testDir)
	if err != nil {
		t.Fatal(err)
	}

	wf := w.WatchedFiles()
	fileList := w.retrieveFileList()

	if len(wf) != len(fileList) {
		t.Fatalf("expected len of wf to be %d, got %d", len(fileList), len(wf))
	}

	// Call close on the watcher even though it's not running.
	w.Close()

	wf = w.WatchedFiles()
	fileList = w.retrieveFileList()

	// Close will be a no-op so there will still be len(fileList) files.
	if len(wf) != len(fileList) {
		t.Fatalf("expected len of wf to be %d, got %d", len(fileList), len(wf))
	}

	// Set running to true.
	w.running = true

	// Now close the watcher.
	go func() {
		// Receive from the w.close channel to avoid a deadlock.
		<-w.close
	}()

	w.Close()

	wf = w.WatchedFiles()

	// Close will be a no-op so there will still be len(fileList) files.
	if len(wf) != 0 {
		t.Fatalf("expected len of wf to be 0, got %d", len(wf))
	}

}

func TestWatchedFiles(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	err := w.Add(testDir)
	if err != nil {
		t.Fatal(err)
	}

	wf := w.WatchedFiles()
	fileList := w.retrieveFileList()

	if len(wf) != len(fileList) {
		t.Fatalf("expected len of wf to be %d, got %d", len(fileList), len(wf))
	}

	for path := range fileList {
		if _, found := wf[path]; !found {
			t.Fatalf("%s not found in watched file's list", path)
		}
	}
}

func TestSetMaxEvents(t *testing.T) {
	w := New()

	if w.maxEvents != 0 {
		t.Fatalf("expected max events to be 0, got %d", w.maxEvents)
	}

	w.SetMaxEvents(3)

	if w.maxEvents != 3 {
		t.Fatalf("expected max events to be 3, got %d", w.maxEvents)
	}
}

func TestOpsString(t *testing.T) {
	testCases := []struct {
		want     Op
		expected string
	}{
		{Create, "CREATE"},
		{Write, "WRITE"},
		{Remove, "REMOVE"},
		{Rename, "RENAME"},
		{Chmod, "CHMOD"},
		{Move, "MOVE"},
		{Op(10), "???"},
	}

	for _, tc := range testCases {
		if tc.want.String() != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, tc.want.String())
		}
	}
}
