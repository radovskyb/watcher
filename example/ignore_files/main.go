package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/radovskyb/watcher"
)

func main() {
	w := watcher.New()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case event := <-w.Event:
				// Print the event.
				fmt.Println(event)
			case err := <-w.Error:
				log.Fatalln(err)
			}
		}
	}()

	// Watch this file for changes.
	if err := w.Add("main.go"); err != nil {
		log.Fatalln(err)
	}

	// Watch test_folder recursively for changes.
	if err := w.Add("../test_folder"); err != nil {
		log.Fatalln(err)
	}

	// Print a list of all of the files and folders currently
	// being watched and their paths.
	for path, f := range w.WatchedFiles() {
		fmt.Printf("%s: %s\n", path, f.Name())
	}
	fmt.Println()

	go func() {
		time.Sleep(time.Second * 3)
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

	wg.Wait()
}
