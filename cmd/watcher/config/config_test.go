package config_test

import (
	"testing"

	"github.com/radovskyb/watcher/cmd/watcher/config"
)

type Defaults = struct {
	interval string
	expected interface{}
}

var c = config.GetConfig()

func TestDefaultsInterval(t *testing.T) {
	if c.Interval != "200ms" {
		t.Error("Failed to set default Interval value")
	}
}

func TestDefaultRecursive(t *testing.T) {
	if !c.Recursive {
		t.Error("Failed to set default Recursive value")
	}
}

func TestDefaultDotFiles(t *testing.T) {
	if !c.Dotfiles {
		t.Error("Failed to set default Dot files value")
	}
}

func TestDefaultCmd(t *testing.T) {
	if c.Cmd != "" {
		t.Error("Failed to set default CMD value")
	}
}

func TestDefaultStartCmd(t *testing.T) {
	if c.Startcmd {
		t.Error("Failed to set default StartCmd value")
	}
}

func TestDefaultListFiles(t *testing.T) {
	if c.ListFiles {
		t.Error("Failed to set default Listfiles value")
	}
}

func TestDefaultStdinPipe(t *testing.T) {
	if c.StdinPipe {
		t.Error("Failed to set default StdinPipe value")
	}
}

func TestDefaultKeepAlive(t *testing.T) {
	if c.Keepalive {
		t.Error("Failed to set default KeepAlive value")
	}
}

func TestDefaultIgnore(t *testing.T) {
	if c.Ignore != "" {
		t.Error("Failed to set default file ignore value")
	}
}
