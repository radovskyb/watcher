# watcher

[![Build Status](https://travis-ci.org/radovskyb/watcher.svg?branch=master)](https://travis-ci.org/radovskyb/watcher)

`watcher` is a simple Go package for watching files or directory changes.

`watcher` watches for changes and notifies over channels either anytime an event or an error has occured.

`watcher`'s simple structure is purposely similar in appearance to fsnotify, yet it doesn't use any system specific events, so should work cross platform consistently.

With `watcher`, when adding a folder to the watchlist, the folder will be watched recursively.

#Installation:

```shell
go get -u github.com/radovskyb/watcher
```

# Todo:

1. Write tests. -- in progress
2. Notify name of file from event. -- done
3. Trigger events. -- done
4. Unique path structures -- done
5. Change options for on individual files/folders add method instead of at initialization?

# Example:

```go
package main

import (
	"fmt"
	"log"
	"sync"

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

	// Watch test_folder recursively for changes.
	if err := w.Add("test_folder"); err != nil {
		log.Fatalln(err)
	}

	// Print a list of all of the files and folders currently
	// being watched and their paths.
	for path, f := range w.Files {
		fmt.Printf("%s: %s\n", path, f.Name())
	}

	fmt.Println()

	// Trigger 2 events after 100 milliseconds.
	go func() {
		time.Sleep(time.Millisecond * 100)
		w.TriggerEvent(watcher.EventFileAdded, nil)
		w.TriggerEvent(watcher.EventFileDeleted, nil)
	}()

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}

	wg.Wait()
}
```
