// +build windows

package watcher

import (
	"os"
	"testing"
)

func TestIsHiddenFileReturnsPathError(t *testing.T) {
	_, err := isHiddenFile("./qqdkqdsdmlqdsd.nop")
	if err == nil {
		t.Fatal("isHidden should have returned an error")
	}

	if _, ok := err.(*os.PathError); !ok {
		t.Fatal("Error is not a PathError")
	}
}