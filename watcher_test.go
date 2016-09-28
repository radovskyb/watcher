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

func TestWatcherAdd(t *testing.T) {
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	// Make sure len(w.Files) is 4.
	if len(w.Files) != 4 {
		t.Errorf("expected 4 files, found %d", len(w.Files))
	}

	// Make sure w.Names[0] is now equal to testDir.
	if w.Names[0] != testDir {
		t.Errorf("expected w.Names[0] to be %s, got %s",
			testDir, w.Names[0])
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

	// Make sure len(w.Files) is 4.
	if len(w.Files) != 4 {
		t.Errorf("expected 4 files, found %d", len(w.Files))
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
	testDir, teardown := setup(t)
	defer teardown()

	w := New()

	// Add the testDir to the watchlist.
	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		select {
		case event := <-w.Event:
			if event.Name() != "triggered event" {
				t.Errorf("expected event file name to be triggered event, got %s",
					event.Name())
			}
			wg.Done()
		case <-time.After(time.Millisecond * 250):
			t.Error("received no event from Event channel")
			wg.Done()
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

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		select {
		case event := <-w.Event:
			if event.EventType != EventFileAdded {
				t.Errorf("expected event to be EventFileAdded, got %s",
					event.EventType)
			}
			if event.Name() != "newfile.txt" {
				t.Errorf("expected event file name to be newfile.txt, got %s",
					event.Name())
			}
			wg.Done()
		case <-time.After(time.Millisecond * 250):
			t.Error("received no event from Event channel")
			wg.Done()
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Error(err)
		}
	}()

	newFileName := filepath.Join(testDir, "newfile.txt")
	err := ioutil.WriteFile(newFileName, []byte{}, 0755)
	if err != nil {
		t.Error(err)
	}

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

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		select {
		case <-w.Event:
			wg.Done()
		case <-time.After(time.Millisecond * 250):
			t.Error("received no event from Event channel")
			wg.Done()
		}
	}()

	go func() {
		// Start the watching process.
		if err := w.Start(time.Millisecond * 100); err != nil {
			t.Error(err)
		}
	}()

	if err := os.Remove(filepath.Join(testDir, "file.txt")); err != nil {
		t.Error(err)
	}

	wg.Wait()
}
