//git-get is like `go get` but for any Git source.
/*
 * Copyright (c) 2014 Will Maier <wcmaier@m.aier.us>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"syscall"
)

var (
	version  string
	fVersion = flag.Bool("version", false, "print version and exit")
)

// usage prints a helpful usage message.
func usage() {
	self := path.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, "usage: %s REPO\n\n", self)
	fmt.Fprint(os.Stderr, "Clone a Git repository, preserving remote structure under GITPATH.\n\n")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Arguments:")
	fmt.Fprintln(os.Stderr, "  REPO     repository to clone")
	fmt.Fprintln(os.Stderr, "Environment variables:")
	fmt.Fprintln(os.Stderr, "  GITPATH  base of local tree of Git clones; defaults to $HOME/src")
	os.Exit(2)
}

// lsRemote calls `git ls-remote --get-url`, resolving a remote to a local path.
func lsRemote(remote string) (string, error) {
	cmd := exec.Command("git", "ls-remote", "--get-url", remote)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out[:])), nil
}

// importPath converts a Git remote path to a local path.
func importPath(remote string) string {
	var (
		h         string
		p         string
		localhost = "localhost/"
	)
	u, _ := url.Parse(remote)
	if u.Host == "" && u.Opaque != "" {
		fields := strings.SplitN(remote, ":", 2)
		h = fields[0]
		p = "/" + fields[1]
	} else if u.Scheme == "" || u.Host == "" {
		fields := strings.SplitN(remote, ":", 2)
		if u.Scheme == "" && len(fields) == 2 && !strings.Contains(fields[0], "/") {
			p = fields[1]
			parts := strings.SplitN(fields[0], "@", 2)
			switch len(parts) {
			case 1:
				h = parts[0]
			case 2:
				h = parts[1]
			}
			h += "/"
		} else {
			h = localhost
			p = u.Path
		}
	} else {
		h = strings.TrimRight(u.Host, ":0123456789")
		p = u.Path
	}
	return path.Clean(h + p)
}

// getGitpath finds a suitable value for GITPATH.
// If the GITPATH environment variable is not set, it defaults to `$HOME/src`.
func getGitpath() string {
	p := os.Getenv("GITPATH")
	if p == "" {
		var home string
		u, err := user.Current()
		if err != nil {
			home = os.Getenv("HOME")
		} else {
			home = u.HomeDir
		}
		p = path.Join(home, "src")
	}
	return p
}

// clone calls `git clone remote local`.
func clone(remote, local string) error {
	cmd := exec.Command("git", "clone", remote, local)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()

	if *fVersion {
		fmt.Printf("git-get %s\n", version)
		os.Exit(0)
	}

	remote := args[0]
	resolved, err := lsRemote(remote)
	if err != nil {
		log.Panic(err)
	}

	gitroot := getGitpath()
	local := path.Join(gitroot, importPath(resolved))

	exit := 0
	err = clone(remote, local)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			exit = waitStatus.ExitStatus()
		} else {
			log.Panic(err)
		}
	}
	os.Exit(exit)
}
