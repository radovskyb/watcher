package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"
	"unicode"

	"github.com/t14/watcher"
	"github.com/t14/watcher/cmd/watcher/config"
)

func main() {
	config := config.GetConfig()
	interval := config.Interval
	recursive := config.Recursive
	dotfiles := config.Dotfiles
	cmd := config.Cmd
	startcmd := config.Startcmd
	listFiles := config.ListFiles
	stdinPipe := config.StdinPipe
	keepalive := config.Keepalive

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
	if cmd != "" {
		split := strings.FieldsFunc(cmd, unicode.IsSpace)
		cmdName = split[0]
		if len(split) > 1 {
			cmdArgs = split[1:]
		}
	}

	// Create a new Watcher with the specified options.
	w := watcher.New()
	w.IgnoreHiddenFiles(!dotfiles)

	done := make(chan struct{})
	go func() {
		defer close(done)

		for {
			select {
			case event := <-w.Event:
				// Print the event's info.
				fmt.Println(event)

				// Run the command if one was specified.
				if cmd != "" {
					c := exec.Command(cmdName, cmdArgs...)
					if stdinPipe {
						c.Stdin = strings.NewReader(event.String())
					} else {
						c.Stdin = os.Stdin
					}
					c.Stdout = os.Stdout
					c.Stderr = os.Stderr
					if err := c.Run(); err != nil {
						if (c.ProcessState == nil || !c.ProcessState.Success()) && keepalive {
							log.Println(err)
							continue
						}
						log.Fatalln(err)
					}
				}
			case err := <-w.Error:
				if err == watcher.ErrWatchedFileDeleted {
					fmt.Println(err)
					continue
				}
				log.Fatalln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	// Add the files and folders specified.
	for _, file := range files {
		if recursive {
			if err := w.AddRecursive(file); err != nil {
				log.Fatalln(err)
			}
		} else {
			if err := w.Add(file); err != nil {
				log.Fatalln(err)
			}
		}
	}

	// Print a list of all of the files and folders being watched.
	if listFiles {
		for path, f := range w.WatchedFiles() {
			fmt.Printf("%s: %s\n", path, f.Name())
		}
		fmt.Println()
	}

	fmt.Printf("Watching %d files\n", len(w.WatchedFiles()))

	// Parse the interval string into a time.Duration.
	parsedInterval, err := time.ParseDuration(interval)
	if err != nil {
		log.Fatalln(err)
	}

	closed := make(chan struct{})

	c := make(chan os.Signal)
	signal.Notify(c, os.Kill, os.Interrupt)
	go func() {
		<-c
		w.Close()
		<-done
		fmt.Println("watcher closed")
		close(closed)
	}()

	// Run the command before watcher starts if one was specified.
	go func() {
		if cmd != "" && startcmd {
			c := exec.Command(cmdName, cmdArgs...)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			if err := c.Run(); err != nil {
				log.Fatalln(err)
			}
		}
	}()

	// Start the watching process.
	if err := w.Start(parsedInterval); err != nil {
		log.Fatalln(err)
	}

	<-closed
}
