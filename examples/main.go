package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/radovskyb/watcher"
)

func main() {
	watcher := watcher.New()

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case event := <-watcher.Event:
				fmt.Println(event)
			case err := <-watcher.Error:
				log.Fatalln(err)
			}
		}
	}()

	// Watch this file for changes.
	if err := watcher.Add("main.go"); err != nil {
		log.Fatalln(err)
	}
	if err := watcher.Add("test_folder"); err != nil {
		log.Fatalln(err)
	}

	// Print a list of all of the file's and folders currently
	// being watched.
	for _, f := range watcher.Files {
		fmt.Println(f.Name())
	}

	// Start the watcher
	if err := watcher.Start(); err != nil {
		log.Fatalln(err)
	}

	wg.Wait()
}
