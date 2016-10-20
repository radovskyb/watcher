package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/radovskyb/watcher"
)

func main() {
	w := watcher.New(watcher.IgnoreDotFiles)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case event := <-w.Event:
				// Print the event's info.
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
				case watcher.EventFileRenamed:
					fmt.Println("Renamed file:", event.Name())
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

	// Watch test_folder recursively for changes.
	//
	// Watcher won't add .dotfile to the watchlist.
	if err := w.Add("../test_folder"); err != nil {
		log.Fatalln(err)
	}

	// Print a list of all of the files and folders currently
	// being watched and their paths.
	for path, f := range w.Files {
		fmt.Printf("%s: %s\n", path, f.Name())
	}

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}

	wg.Wait()
}
