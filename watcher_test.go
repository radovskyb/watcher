package watcher

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// setup creates all required files and folders for
// the tests and returns a function that is used as
// a teardown function when the tests are done.
func setup(t *testing.T) (string, func()) {
	testDir, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Error(err)
	}

	err = ioutil.WriteFile(filepath.Join(testDir, "file.txt"),
		[]byte{}, 0755)
	if err != nil {
		t.Error(err)
	}

	files := []string{"file_1.txt", "file_2.txt", "file_3.txt"}

	for _, f := range files {
		filePath := filepath.Join(testDir, f)
		if err := ioutil.WriteFile(filePath, []byte{}, 0755); err != nil {
			t.Error(err)
		}
	}

	err = ioutil.WriteFile(filepath.Join(testDir, ".dotfile"),
		[]byte{}, 0755)
	if err != nil {
		t.Error(err)
	}

	testDirTwo := filepath.Join(testDir, "testDirTwo")
	err = os.Mkdir(testDirTwo, 0755)
	if err != nil {
		t.Error(err)
	}

	err = ioutil.WriteFile(filepath.Join(testDirTwo, "file_recursive.txt"),
		[]byte{}, 0755)
	if err != nil {
		t.Error(err)
	}

	return testDir, func() {
		if os.RemoveAll(testDir); err != nil {
			t.Error(err)
		}
	}
}

func TestSetNonRecursive(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New(NonRecursive)

	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	if len(w.Files) != 7 {
		t.Errorf("expected len(w.Files) to be 7, got %d", len(w.Files))
	}

	// Make sure w.Names[0] is now equal to testDir.
	if w.Names[0] != testDir {
		t.Errorf("expected w.Names[0] to be %s, got %s",
			testDir, w.Names[0])
	}

	if _, found := w.Files[testDir]; !found {
		t.Errorf("expected to find %s", testDir)
	}

	if w.Files[testDir].Name() != testDir {
		t.Errorf("expected w.Files[%q].Name() to be %s, got %s",
			testDir, testDir, w.Files[testDir].Name())
	}

	dotFile := filepath.Join(testDir, ".dotfile")
	if _, found := w.Files[dotFile]; !found {
		t.Errorf("expected to find %s", dotFile)
	}

	if w.Files[dotFile].Name() != ".dotfile" {
		t.Errorf("expected w.Files[%q].Name() to be .dotfile, got %s",
			dotFile, w.Files[dotFile].Name())
	}

	fileRecursive := filepath.Join(testDir, "testDirTwo", "file_recursive.txt")
	if _, found := w.Files[fileRecursive]; found {
		t.Errorf("expected to not find %s", fileRecursive)
	}

	fileTxt := filepath.Join(testDir, "file.txt")
	if _, found := w.Files[fileTxt]; !found {
		t.Errorf("expected to find %s", fileTxt)
	}

	if w.Files[fileTxt].Name() != "file.txt" {
		t.Errorf("expected w.Files[%q].Name() to be file.txt, got %s",
			fileTxt, w.Files[fileTxt].Name())
	}

	dirTwo := filepath.Join(testDir, "testDirTwo")
	if _, found := w.Files[dirTwo]; !found {
		t.Errorf("expected to find %s directory", dirTwo)
	}

	if w.Files[dirTwo].Name() != "testDirTwo" {
		t.Errorf("expected w.Files[%q].Name() to be testDirTwo, got %s",
			dirTwo, w.Files[dirTwo].Name())
	}
}

func TestSetIgnoreDotFiles(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New(IgnoreDotFiles)

	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	if len(w.Files) != 7 {
		t.Errorf("expected len(w.Files) to be 7, got %d", len(w.Files))
	}

	// Make sure w.Names[0] is now equal to testDir.
	if w.Names[0] != testDir {
		t.Errorf("expected w.Names[0] to be %s, got %s",
			testDir, w.Names[0])
	}

	if _, found := w.Files[testDir]; !found {
		t.Errorf("expected to find %s", testDir)
	}

	if w.Files[testDir].Name() != testDir {
		t.Errorf("expected w.Files[%q].Name() to be %s, got %s",
			testDir, testDir, w.Files[testDir].Name())
	}

	fileRecursive := filepath.Join(testDir, "testDirTwo", "file_recursive.txt")
	if _, found := w.Files[fileRecursive]; !found {
		t.Errorf("expected to find %s", fileRecursive)
	}

	if _, found := w.Files[filepath.Join(testDir, ".dotfile")]; found {
		t.Error("expected to not find .dotfile")
	}

	fileTxt := filepath.Join(testDir, "file.txt")
	if _, found := w.Files[fileTxt]; !found {
		t.Errorf("expected to find %s", fileTxt)
	}

	if w.Files[fileTxt].Name() != "file.txt" {
		t.Errorf("expected w.Files[%q].Name() to be file.txt, got %s",
			fileTxt, w.Files[fileTxt].Name())
	}

	dirTwo := filepath.Join(testDir, "testDirTwo")
	if _, found := w.Files[dirTwo]; !found {
		t.Errorf("expected to find %s directory", dirTwo)
	}

	if w.Files[dirTwo].Name() != "testDirTwo" {
		t.Errorf("expected w.Files[%q].Name() to be testDirTwo, got %s",
			dirTwo, w.Files[dirTwo].Name())
	}
}

func TestSetIgnoreDotFilesAndNonRecursive(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New(IgnoreDotFiles, NonRecursive)

	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	if len(w.Files) != 6 {
		t.Errorf("expected len(w.Files) to be 6, got %d", len(w.Files))
	}

	// Make sure w.Names[0] is now equal to testDir.
	if w.Names[0] != testDir {
		t.Errorf("expected w.Names[0] to be %s, got %s",
			testDir, w.Names[0])
	}

	if _, found := w.Files[testDir]; !found {
		t.Errorf("expected to find %s", testDir)
	}

	if w.Files[testDir].Name() != testDir {
		t.Errorf("expected w.Files[%q].Name() to be %s, got %s",
			testDir, testDir, w.Files[testDir].Name())
	}

	if _, found := w.Files[filepath.Join(testDir, ".dotfile")]; found {
		t.Error("expected to not find .dotfile")
	}

	fileRecursive := filepath.Join(testDir, "testDirTwo", "file_recursive.txt")
	if _, found := w.Files[fileRecursive]; found {
		t.Errorf("expected to not find %s", fileRecursive)
	}

	fileTxt := filepath.Join(testDir, "file.txt")
	if _, found := w.Files[fileTxt]; !found {
		t.Errorf("expected to find %s", fileTxt)
	}

	if w.Files[fileTxt].Name() != "file.txt" {
		t.Errorf("expected w.Files[%q].Name() to be file.txt, got %s",
			fileTxt, w.Files[fileTxt].Name())
	}

	dirTwo := filepath.Join(testDir, "testDirTwo")
	if _, found := w.Files[dirTwo]; !found {
		t.Errorf("expected to find %s directory", dirTwo)
	}

	if w.Files[dirTwo].Name() != "testDirTwo" {
		t.Errorf("expected w.Files[%q].Name() to be testDirTwo, got %s",
			dirTwo, w.Files[dirTwo].Name())
	}
}

func TestWatcherAdd(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	// Make sure len(w.Files) is 8.
	if len(w.Files) != 8 {
		t.Errorf("expected 8 files, found %d", len(w.Files))
	}

	// Make sure w.Names[0] is now equal to testDir.
	if w.Names[0] != testDir {
		t.Errorf("expected w.Names[0] to be %s, got %s",
			testDir, w.Names[0])
	}

	dirTwo := filepath.Join(testDir, "testDirTwo")
	if _, found := w.Files[dirTwo]; !found {
		t.Errorf("expected to find %s directory", dirTwo)
	}

	if w.Files[dirTwo].Name() != "testDirTwo" {
		t.Errorf("expected w.Files[%q].Name() to be testDirTwo, got %s",
			"testDirTwo", w.Files[dirTwo].Name())
	}

	fileRecursive := filepath.Join(dirTwo, "file_recursive.txt")
	if _, found := w.Files[fileRecursive]; !found {
		t.Errorf("expected to find %s directory", fileRecursive)
	}

	if w.Files[fileRecursive].Name() != "file_recursive.txt" {
		t.Errorf("expected w.Files[%q].Name() to be file_recursive.txt, got %s",
			fileRecursive, w.Files[fileRecursive].Name())
	}
}

func TestWatcherAddNotFound(t *testing.T) {
	w := New()

	// Make sure there is an error when adding a
	// non-existent file/folder.
	if err := w.Add("random_filename.txt"); err == nil {
		t.Error("expected a file not found error")
	}
}

func TestWatcherRemove(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	// Add the testDir to the watchlist.
	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	// Make sure len(w.Files) is 8.
	if len(w.Files) != 8 {
		t.Errorf("expected 8 files, found %d", len(w.Files))
	}

	// Now remove the folder from the watchlist.
	if err := w.Remove(testDir); err != nil {
		t.Error(err)
	}

	// Now check that there is nothing being watched.
	if len(w.Files) != 0 {
		t.Errorf("expected len(w.Files) to be 0, got %d", len(w.Files))
	}

	// Make sure len(w.Names) is now 0.
	if len(w.Names) != 0 {
		t.Errorf("expected len(w.Names) to be empty, len(w.Names): %s", len(w.Names))
	}
}

func TestListFiles(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	fileList, err := ListFiles(testDir)
	if err != nil {
		t.Error(err)
	}

	// Make sure fInfoTest contains the correct os.FileInfo names.
	fname := filepath.Join(testDir, "file.txt")
	if fileList[fname].Name() != "file.txt" {
		t.Errorf("expected fileList[%s].Name() to be file.txt, got %s",
			fname, fileList[fname].Name())
	}
}

func TestTriggerEvent(t *testing.T) {
	w := New()

	// Add the testDir to the watchlist.
	if err := w.Add("watcher_test.go"); err != nil {
		t.Error(err)
	}

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
			t.Error("received no event from Event channel")
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Error(err)
		}
	}()

	w.TriggerEvent(EventFileAdded, nil)

	wg.Wait()
}

func TestEventAddFile(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	// Add the testDir to the watchlist.
	if err := w.Add(testDir); err != nil {
		t.Error(err)
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
				if event.EventType != EventFileAdded {
					t.Errorf("expected event to be EventFileAdded, got %s",
						event.EventType)
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
			}
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()
}

func TestEventDeleteFile(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	// Add the testDir to the watchlist.
	if err := w.Add(testDir); err != nil {
		t.Error(err)
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
				if event.EventType != EventFileDeleted {
					t.Errorf("expected event to be EventEventFileDeleted, got %s",
						event.EventType)
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
			}
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()
}
