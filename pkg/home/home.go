package home

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/tychoish/fun/adt"
)

var cache *adt.Once[string]

func init() {
	cache = &adt.Once[string]{}
	cache.Set(func() string {
		if runtime.GOOS == "windows" {
			if dir := os.Getenv("HOME"); dir != "" {
				return dir
			} else if dir := os.Getenv("USERPROFILE"); dir != "" {
				return dir
			}

			drive := os.Getenv("HOMEDRIVE")
			path := os.Getenv("HOMEPATH")
			if drive != "" && path != "" {
				return fmt.Sprint(drive, path)
			}
			return ""
		}
		var envVar string
		if runtime.GOOS == "plan9" {
			envVar = "home"
		} else {
			envVar = "HOME"
		}

		if dir := os.Getenv(envVar); dir != "" {
			return dir
		}

		cmd := exec.Command("sh", "-c", "cd && pwd")
		out, err := cmd.Output()
		out = bytes.TrimSpace(out)
		if err != nil || len(out) == 0 {
			return "UNKNOWN_HOMEDIR"
		}

		return string(out)
	})
}

func GetDirectory() string { return cache.Resolve() }

func TryExpandDirectory(in string) string {
	if len(in) == 0 {
		return ""
	}

	if in[0] != '~' {
		return in
	}

	if len(in) > 1 && in[1] != '/' && in[1] != '\\' {
		// these are "~foo" or "~\" values which are ambiguous
		return in
	}

	return filepath.Join(GetDirectory(), in[1:])
}

func TryCollapseDirectory(in string) string {
	hd := GetDirectory()
	if strings.HasPrefix(in, hd) {
		return strings.Replace(in, hd, "~", 1)
	}
	return in
}
