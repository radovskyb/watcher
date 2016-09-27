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

	// Start the watcher
	if err := w.Start(); err != nil {
		log.Fatalln(err)
	}

	wg.Wait()
}
