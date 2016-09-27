# watcher
`watcher` is a simple Go package for watching for file/file's or directory/multiple directory changes.

`watcher` is a watcher that watches for changes and notifies over channel's either anytime an event or an error has occured.

`watcher`'s simple structure is purposely similar in appearance to fsnotify, yet doesn't use any system specific events, so should work cross platform consistently.

With `watcher`, when adding a folder to the watchlist, the folder will be watched recursively.

# Todo:

1. Write tests.
2. Notify name of file from event.

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

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case event := <-w.Event:
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
	if err := w.Add("test_folder"); err != nil {
		log.Fatalln(err)
	}

	// Print a list of all of the file's and folders currently
	// being watched.
	for _, f := range w.Files {
		fmt.Println(f.Name())
	}

	// Start the watcher to check for changes every 100ms.
	if err := w.Start(100); err != nil {
		log.Fatalln(err)
	}

	wg.Wait()
}
```
