package main

import (
	"fmt"
	"log"
	"time"

	"github.com/radovskyb/watcher"
)

func main() {
	w := watcher.New()

	// SetMaxEvents to 2 to allow at most 2 event's to be received
	// on the Event channel per watching cycle.
	//
	// If SetMaxEvents is not set, the default is to send all events.
	w.SetMaxEvents(2)

	go func() {
		for {
			select {
			case event := <-w.Event:
				// Print the event's info.
				fmt.Println(event)

				// Print out the file name with a message
				// based on the event type.
				switch event.Op {
				case watcher.Write:
					fmt.Println("Wrote file:", event.Name())
				case watcher.Create:
					fmt.Println("Created file:", event.Name())
				case watcher.Remove:
					fmt.Println("Removed file:", event.Name())
				case watcher.Rename:
					fmt.Println("Renamed file:", event.Name())
				case watcher.Chmod:
					fmt.Println("Chmoded file:", event.Name())
				}
			case err := <-w.Error:
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	// Watch this folder for changes.
	if err := w.Add("."); err != nil {
		log.Fatalln(err)
	}

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

	// Trigger 2 events after 100 milliseconds.
	go func() {
		time.Sleep(time.Millisecond * 100)
		w.TriggerEvent(watcher.Create, nil)
		w.TriggerEvent(watcher.Remove, nil)
	}()

	// Start the watching process - it'll check for changes every 100ms.
	if err := w.Start(time.Millisecond * 100); err != nil {
		log.Fatalln(err)
	}
}
