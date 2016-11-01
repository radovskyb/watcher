package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/radovskyb/watcher"
)

func main() {
	interval := flag.String("interval", "100ms", "watcher poll interval")
	recursive := flag.Bool("recursive", true, "watch folders recursively")
	dotfiles := flag.Bool("dotfiles", true, "watch dot files")
	cmd := flag.String("cmd", "", "command to run when an event occurs")
	listFiles := flag.Bool("list", false, "list watched files on start")
	stdinPipe := flag.Bool("pipe", false, "pipe event's info to command's stdin")

	flag.Parse()

	// Retrieve the list of files and folders.
	files := flag.Args()

	// If no files/folders were specified, watch the current directory.
	if len(files) == 0 {
		curDir, err := os.Getwd()
		if err != nil {
			log.Fatalln(err)
		}
		files = append(files, curDir)
	}

	var cmdName string
	var cmdArgs []string
	if *cmd != "" {
		split := strings.FieldsFunc(*cmd, unicode.IsSpace)
		cmdName = split[0]
		if len(split) > 1 {
			cmdArgs = split[1:]
		}
	}

	options := []watcher.Option{}

	// If recursive is set to false, set watcher.NonRecursive.
	if !*recursive {
		options = append(options, watcher.NonRecursive)
	}

	// If dotfiles is set to false, set watcher.IgnoreDotFiles.
	if !*dotfiles {
		options = append(options, watcher.IgnoreDotFiles)
	}

	// Create a new Watcher with the specified options.
	w := watcher.New(options...)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case event := <-w.Event:
				// Print the event's info.
				fmt.Println(event)

				// Run the command if one was specified.
				if *cmd != "" {
					c := exec.Command(cmdName, cmdArgs...)
					if *stdinPipe {
						c.Stdin = strings.NewReader(event.String())
					} else {
						c.Stdin = os.Stdin
					}
					c.Stdout = os.Stdout
					c.Stderr = os.Stderr
					if err := c.Run(); err != nil {
						log.Fatalln(err)
					}
				}
			case err := <-w.Error:
				log.Fatalln(err)
			}
		}
	}()

	// Add the files and folders specified.
	for _, file := range files {
		if err := w.Add(file); err != nil {
			log.Fatalln(err)
		}
	}

	// Print a list of all of the files and folders being watched.
	if *listFiles {
		for path, f := range w.WatchedFiles() {
			fmt.Printf("%s: %s\n", path, f.Name())
		}
		fmt.Println()
	}

	fmt.Printf("Watching %d files\n", len(w.WatchedFiles()))

	// Parse the interval string into a time.Duration.
	parsedInterval, err := time.ParseDuration(*interval)
	if err != nil {
		log.Fatalln(err)
	}

	// Start the watching process.
	if err := w.Start(parsedInterval); err != nil {
		log.Fatalln(err)
	}

	wg.Wait()
}
