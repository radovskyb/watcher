package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/radovskyb/watcher"
)

func main() {
	// There are 2 ways to use both options at the same time.
	//
	// 1) By comma separating the 2 arguments.
	w := watcher.New(watcher.NonRecursive, watcher.IgnoreDotFiles)

	// 2) By ORing the 2 options together.
	// w := watcher.New(watcher.NonRecursive | watcher.IgnoreDotFiles)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case event := <-w.Event:
				// Print the event type.
				fmt.Println(event)

				// Print out the file name with a message
				// based on the event type.
				switch event.EventType {
				case watcher.EventFileModified:
					fmt.Println("Modified file:", event.Name())
				case watcher.EventFileAdded:
					fmt.Println("Added file:", event.Name())
				case watcher.EventFileDeleted:
					fmt.Println("Deleted file:", event.Name())
				}
			case err := <-w.Error:
				log.Fatalln(err)
			}
		}
	}()

	// Watch this file for changes.
	if err := w.Add("main.go"); err != nil {
		log.Fatalln(err)
	}

	// Watch test_folder non-recursively for changes.
	//
	// Watcher won't add the file test_folder_recursive/file_recursive.txt or the file .dotfile.
	if err := w.Add("../test_folder"); err != nil {
		log.Fatalln(err)
	}

	// Print a list of all of the files and folders currently
	// being watched.
	for path, f := range w.Files {
		fmt.Printf("%s: %s\n", path, f.Name())
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}

	wg.Wait()
}
