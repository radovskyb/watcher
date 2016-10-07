# watcher command

# Installation

```shell
go get -u github.com/radovskyb/watcher/...
```

# Usage

```
Usage of watcher:
  -cmd string
    	command to run when an event occurs
  -dotfiles
    	watch dot files (default true)
  -interval string
    	watcher poll interval (default "100ms")
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
