package watcher

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const testDir = "examples/test_folder"

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
	fInfoList, err := ListFiles(testDir)
	if err != nil {
		t.Error(err)
	}

	// Make sure fInfoTest contains the correct os.FileInfo names.
	if fInfoList[0].Name() != filepath.Base(testDir) {
		t.Errorf("expected fInfoList[0].Name() to be test_folder, got %s",
			fInfoList[0].Name())
	}
	if fInfoList[1].Name() != "file.txt" {
		t.Errorf("expected fInfoList[1].Name() to be file.txt, got %s",
			fInfoList[1].Name())
	}
}

func TestEventAddFile(t *testing.T) {
	w := New()

	// Add the testDir to the watchlist.
	if err := w.Add(testDir); err != nil {
		t.Error(err)
	}

	go func() {
		// Start the watching process.
		if err := w.Start(100); err != nil {
			t.Error(err)
		}
	}()

	go func() {
		time.Sleep(time.Millisecond * 10)
		newFileName := filepath.Join(testDir, "newfile.txt")
		err := ioutil.WriteFile(newFileName, []byte("Hello, World!"), os.ModePerm)
		if err != nil {
			t.Error(err)
		}
		if err := os.Remove(newFileName); err != nil {
			t.Error(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	select {
	case <-w.Event:
		wg.Done()
	case <-time.After(time.Millisecond * 250):
		t.Error("received no event from Event channel")
		wg.Done()
	}

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
		// Start the watching process.
		if err := w.Start(100); err != nil {
			t.Error(err)
		}
	}()

	go func() {
		time.Sleep(time.Millisecond * 10)
		if err := os.Remove(fileName); err != nil {
			t.Error(err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	select {
	case <-w.Event:
		wg.Done()
	case <-time.After(time.Millisecond * 250):
		t.Error("received no event from Event channel")
		wg.Done()
	}

	wg.Wait()
}
