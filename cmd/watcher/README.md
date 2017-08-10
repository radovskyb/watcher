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
  -keepalive
    	keep alive when a cmd returns code != 0
  -list
    	list watched files on start
  -pipe
    	pipe event's info to command's stdin
  -recursive
    	watch folders recursively (default true)
  -startcmd
    	run the command when watcher starts
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
