package main

import (
	"fmt"
	"log"
	"time"

	"github.com/radovskyb/watcher"
)

func main() {
	w := watcher.New()

	go func() {
		for {
			select {
			case event := <-w.Event:
				// Print the event.
				fmt.Println(event)
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	// Watch test_folder recursively for changes.
	if err := w.AddRecursive("../test_folder"); err != nil {
		log.Fatalln(err)
	}

	// Print a list of all of the files and folders currently
	// being watched and their paths.
	for path, f := range w.WatchedFiles() {
		fmt.Printf("%s: %s\n", path, f.Name())
	}
	fmt.Println()

	go func() {
		w.Wait()
		// Ignore ../test_folder/test_folder_recursive and ../test_folder/.dotfile
		if err := w.Ignore("../test_folder/test_folder_recursive", "../test_folder/.dotfile"); err != nil {
			log.Fatalln(err)
		}
		// Print a list of all of the files and folders currently being watched
		// and their paths after adding files and folders to the ignore list.
		for path, f := range w.WatchedFiles() {
			fmt.Printf("%s: %s\n", path, f.Name())
		}
		fmt.Println()
	}()

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}
