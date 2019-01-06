package config

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
)

// Config values that can be used in a json file.
type Config struct {
	Interval  string `json:"interval"`
	Recursive bool   `json:"recursive"`
	Dotfiles  bool   `json:"dotfiles"`
	Cmd       string `json:"cmd"`
	Startcmd  bool   `json:"startcmd"`
	ListFiles bool   `json:"listfiles"`
	StdinPipe bool   `json:"pipe"`
	Keepalive bool   `json:"keepalive"`
}

var c Config

// GetConfig Checks for a user defined config file. If there is no config file the
// command line flags or their defaults are used.
func GetConfig() Config {
	configFile := flag.String("config", "", "config file")
	interval := flag.String("interval", "100ms", "watcher poll interval")
	recursive := flag.Bool("recursive", true, "watch folders recursively")
	dotfiles := flag.Bool("dotfiles", true, "watch dot files")
	cmd := flag.String("cmd", "", "command to run when an event occurs")
	startcmd := flag.Bool("startcmd", false, "run the command when watcher starts")
	listFiles := flag.Bool("list", false, "list watched files on start")
	stdinPipe := flag.Bool("pipe", false, "pipe event's info to command's stdin")
	keepalive := flag.Bool("keepalive", false, "keep alive when a cmd returns code != 0")

	flag.Parse()

	// If the configfile flag is defined use its values.
	if len(*configFile) > 0 {
		jsonFile, err := os.Open(*configFile)
		if err != nil {
			panic(err)
		}

		defer jsonFile.Close()

		byteValue, error := ioutil.ReadAll(jsonFile)
		if error != nil {
			panic(error)
		}

		var data Config
		json.Unmarshal(byteValue, &data)
		// Set config values. If a value is not set in the config file
		// the default value will be used.
		c.setInterval(data.Interval)
		c.setRecursive(data.Recursive)
		c.setDotfiles(data.Dotfiles)
		c.setCmd(data.Cmd)
		c.setStartCmd(data.Startcmd)
		c.setListFiles(data.ListFiles)
		c.setPipe(data.StdinPipe)
		c.setKeepAlive(data.Keepalive)

	} else {
		//Use CLI values
		c.Interval = *interval
		c.Recursive = *recursive
		c.Dotfiles = *dotfiles
		c.Cmd = *cmd
		c.Startcmd = *startcmd
		c.ListFiles = *listFiles
		c.StdinPipe = *stdinPipe
		c.Keepalive = *keepalive
	}

	return c
}

func (c *Config) setInterval(interval string) {
	c.Interval = "200ms"
	if len(interval) > 0 {
		c.Interval = interval
	}
}

func (c *Config) setRecursive(recursive bool) {
	c.Recursive = true
	if !recursive {
		c.Recursive = false
	}
}

func (c *Config) setDotfiles(dotfiles bool) {
	c.Dotfiles = true
	if !dotfiles {
		c.Dotfiles = false
	}
}

func (c *Config) setCmd(cmd string) {
	c.Cmd = ""
	if len(cmd) > 0 {
		c.Cmd = cmd
	}
}

func (c *Config) setStartCmd(startCmd bool) {
	c.Startcmd = false
	if startCmd {
		c.Startcmd = true
	}
}

func (c *Config) setListFiles(listFiles bool) {
	c.ListFiles = false
	if listFiles {
		c.ListFiles = true
	}
}

func (c *Config) setPipe(pipe bool) {
	c.StdinPipe = false
	if pipe {
		c.StdinPipe = true
	}
}

func (c *Config) setKeepAlive(keepAlive bool) {
	c.Keepalive = false
	if keepAlive {
		c.Keepalive = true
	}
}
