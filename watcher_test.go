package watcher

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const testDir = "example/test_folder"

func TestWatcherAdd(t *testing.T) {
	w := New()

	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	// Make sure w.Files is the right amount, including
	// file and folders.
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
	w := New()

	// Add the testDir to the watchlist.
	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	// Make sure w.Files is the right amount, including
	// file and folders.
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
	fileList, err := ListFiles(testDir)
	if err != nil {
		t.Error(err)
	}

	// Make sure fInfoTest contains the correct os.FileInfo names.
	if fileList["test_folder/file.txt"].Name() != "file.txt" {
		t.Errorf("expected fileList[\"file.txt\"].Name() to be file.txt, got %s",
			fileList["test_folder/file.txt"].Name())
	}
}

func TestTriggerEvent(t *testing.T) {
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
		if err := w.Start(100); err != nil {
			t.Error(err)
		}
	}()

	w.TriggerEvent(EventFileAdded, nil)

	wg.Wait()
}

func TestEventAddFile(t *testing.T) {
	w := New()

	// Add the testDir to the watchlist.
	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	newFileName := filepath.Join(testDir, "newfile.txt")
	err := ioutil.WriteFile(newFileName, []byte("Hello, World!"), os.ModePerm)
	if err != nil {
		t.Error(err)
	}
	if err := os.Remove(newFileName); err != nil {
		t.Error(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		select {
		case event := <-w.Event:
			// TODO: Make event's accurate where if a modified event is a file,
			// don't return the file's folder first as a modified folder.
			//
			// Will be modified event because the folder will be checked first.
			if event.EventType != EventFileModified {
				t.Errorf("expected event to be EventFileModified, got %s",
					event.EventType)
			}
			// For the same reason as above, the modified file won't be newfile.txt,
			// but rather test_folder.
			if event.Name() != "test_folder" {
				t.Errorf("expected event file name to be test_folder, got %s",
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
		if err := w.Start(100); err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()
}

func TestEventDeleteFile(t *testing.T) {
	fileName := filepath.Join(testDir, "file.txt")

	// Put the file back when the test is finished.
	defer func() {
		f, err := os.Create(fileName)
		if err != nil {
			t.Error(err)
		}
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()

	w := New()

	// Add the testDir to the watchlist.
	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	go func() {
		if err := os.Remove(fileName); err != nil {
			t.Error(err)
		}
	}()

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
		if err := w.Start(0); err != nil {
			t.Error(err)
		}
	}()

	wg.Wait()
}
