# watcher

[![Build Status](https://travis-ci.org/radovskyb/watcher.svg?branch=master)](https://travis-ci.org/radovskyb/watcher)

`watcher` is a simple Go package for watching for files or directory changes (recursively by default) without using filesystem events, which allows it to work cross platform consistently.

`watcher` watches for changes and notifies over channels either anytime an event or an error has occured.

Events contain the `os.FileInfo` of the file or directory that the event is based on and the type of event that has occured.

# Installation

```shell
go get -u github.com/radovskyb/watcher/...
```

# Contributing
If you would ike to contribute, simply submit a pull request.

# Features

- Customizable polling interval.
- Watch folders recursively or non-recursively.
- Choose to ignore dot files.
- Notify the `os.FileInfo` of the file that the event is based on. e.g `Name`, `ModTime`, `IsDir`, etc.
- Notify the full path of the file that the event is based on.
- Trigger custom events.
- Limit amount of events that can be received per watching cycle.
- Choose to list the files being watched.
- Ignore specific files and folders.

# Todo

- Write more tests.
- Write benchmarks.
- Watch only specific extensions. (yes/no/maybe?)
- Make sure renames based on modtime actually work cross platform.

# Command

`watcher` comes with a simple command which is installed when using the `go get` command from above.

# Usage

```
Usage of watcher:
  -cmd string
    	command to run when an event occurs
  -dotfiles
    	watch dot files (default true)
  -interval string
    	watcher poll interval (default "100ms")
  -list 
    	list watched files on start (default false)
  -pipe
    	pipe event's info to command's stdin (default false)
  -recursive
    	watch folders recursively (default true)
```

All of the flags are optional and watcher can be simply called by itself:
```shell
watcher
```
(watches the current directory recursively for changes and notifies for any events that occur.)

A more elaborate example using the `watcher` command:
```shell
watcher -dotfiles=false -recursive=false -cmd="./myscript" main.go ../
```
In this example, `watcher` will ignore dot files and folders and won't watch any of the specified folders recursively. It will also run the script `./myscript` anytime an event occurs while watching `main.go` or any files or folders in the previous directory (`../`).

Using the `pipe` and `cmd` flags together will send the event's info to the command's stdin when changes are detected.

First create a file called `script.py` with the following contents:
```python
import sys

for line in sys.stdin:
	print (line + " - python")
```

Next, start watcher with the `pipe` and `cmd` flags enabled:
```shell
watcher -cmd="python script.py" -pipe=true
```

Now when changes are detected, the event's info will be output from the running python script.

# Example

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

	// SetMaxEvents to 1 to allow at most 1 Event to be received
	// on the Event channel per watching cycle.
	//
	// If SetMaxEvents is not set, the default is to send all events.
	w.SetMaxEvents(1)

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

	wg.Wait()
}
```
