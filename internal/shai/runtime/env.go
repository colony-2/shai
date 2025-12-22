package shai

import (
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
)

func hostEnvMap() map[string]string {
	env := make(map[string]string)
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		env[parts[0]] = parts[1]
	}
	return env
}

// hostUserIDs returns the host UID and GID as strings, falling back to "4747"
// when the detected UID is 0 to avoid running the sandbox as root by default.
func hostUserIDs() (uid string, gid string) {
	uid = "4747"
	gid = "4747"

	if u := os.Getuid(); u > 0 {
		uid = strconv.Itoa(u)
	}
	if g := os.Getgid(); g > 0 {
		gid = strconv.Itoa(g)
	}

	// Fallback to os/user when available (useful on platforms where Getuid
	// returns 0 but the process is actually a non-root user).
	if (uid == "4747" || gid == "4747") && runtime.GOOS != "windows" {
		if current, err := user.Current(); err == nil {
			if current.Uid != "" && current.Uid != "0" {
				uid = current.Uid
			}
			if current.Gid != "" && current.Gid != "0" {
				gid = current.Gid
			}
		}
	}

	// Keep the fallback when the host user really is root.
	if uid == "0" {
		uid = "4747"
	}
	if gid == "0" {
		gid = "4747"
	}
	return uid, gid
}
