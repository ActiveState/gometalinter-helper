package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

func main() {
	flag := ""
	if len(os.Args) > 1 {
		flag = os.Args[1]
	}

	var files []string
	if regexp.MustCompile(`^--?h(?:elp)?`).MatchString(flag) {
		usage()
		os.Exit(0)
	} else if flag == "-all" {
		files = allGoFiles()
	} else if flag == "-commit-hook" {
		files = commitHookGoFiles()
	} else if os.Getenv("CI") != "" {
		files = ciGoFiles()
	} else {
		files = os.Args[1:]
	}

	if len(files) == 0 {
		os.Exit(0)
	}

	sort.Strings(files)
	os.Exit(checkFiles(files))
}

func usage() {
	fmt.Print(`
  check-go-files ...

  This command can be used to check Go file for conformance with various
  tools. It is intended to be used both as part of your Git commit hooks and as
  a tool to be run under CI. You can also run it as a standalone command.

  It has a number of modes, depending on what arguments you pass it and the
  presence of a CI environment variable.

  If you pass the "-all" flag then it will check all Go files in the current
  directory tree.

  If you pass the "-commit-hook" flag then it will check new or modified files
  that are about to be committed. 

  If the neither flag is passed and the CI environment variable is set, then it
  will run a check of the current branch. If that branch is "master" than it
  checks all Go files (like the "-all" flag). Otherwise it checks Go files in
  the current branch that differ from master.

  Finally, you can pass an explicit list of files to check.

  It executes the following programs to do these checks:

  * gofmt
  * golint
  * go vet
  * errcheck
  * dep - only if a Gopkg.lock file is present

`)
}

func allGoFiles() []string {
	var files []string
	err := filepath.Walk(".", func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == ".git" || path == "vendor" {
			return filepath.SkipDir
		}

		if isGoFile(path) {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Error walking tree: %v", err)
	}

	return files
}

func commitHookGoFiles() []string {
	output := stdoutFrom("git", "diff", "--cached", "--name-only", "--diff-filter=ACM")
	return linesFrom(output)
}

func ciGoFiles() []string {
	branch := strings.TrimSpace(stdoutFrom("git", "rev-parse", "--abbrev-ref", "HEAD"))
	if branch == "master" {
		return allGoFiles()
	}
	return linesFrom(stdoutFrom("git", "diff", "--name-only", "master..."+branch))
}

func stdoutFrom(args ...string) string {
	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error running %s: %v\nStderr:\n%s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

func linesFrom(from string) []string {
	s := bufio.NewScanner(strings.NewReader(from))
	var lines []string
	for s.Scan() {
		if isGoFile(s.Text()) {
			lines = append(lines, s.Text())
		}
	}
	return lines
}

func isGoFile(path string) bool {
	return regexp.MustCompile(`\.go$`).MatchString(path)
}

func checkFiles(files []string) int {
	exit := 0
	pkgs := dirs(files)

	exit += gofmt(files)
	exit += golint(pkgs)
	exit += vet(pkgs)
	exit += errcheck(pkgs)
	exit += dep()

	return exit
}

func dirs(files []string) []string {
	m := make(map[string]bool)
	for _, f := range files {
		m[filepath.Join(filepath.Dir(f))] = true
	}
	var dirs []string
	for d := range m {
		// We need to prefix ./ to the directory paths so that these packages
		// can be found. Otherwise it looks for $GOPATH/$pkg which will not
		// work at all.
		dirs = append(dirs, "./"+d)
	}

	sort.Strings(dirs)
	return dirs
}

func gofmt(files []string) int {
	c := []string{"gofmt", "-l"}
	c = append(c, files...)
	return commandStatus(c, "gofmt", true, false)
}

func golint(pkgs []string) int {
	c := []string{"golint"}
	c = append(c, pkgs...)
	return commandStatus(c, "golint", true, false)
}

func vet(pkgs []string) int {
	c := []string{"go", "vet"}
	c = append(c, pkgs...)
	return commandStatus(c, "go vet", false, true)
}

func errcheck(pkgs []string) int {
	c := []string{"errcheck"}
	c = append(c, pkgs...)
	return commandStatus(c, "errcheck", false, false)
}

func dep() int {
	if !fileExists("Gopkg.lock") {
		return 0
	}

	return commandStatus([]string{"dep", "status"}, "dep status", false, true)
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

func commandStatus(c []string, what string, checkStdout bool, useStderr bool) int {
	o, e, s := run(c...)
	pass := o
	if useStderr {
		pass = e
	}

	// Some commands indicate failure simply by printing to stdout. Others may
	// print output but the exit code is what indicates failure.
	if (checkStdout && o != "") || (s == 1 && !checkStdout) {
		pass = regexp.MustCompile(`(?m)^`).ReplaceAllLiteralString(pass, "  ")
		msg := `
Go files must pass %s:

%s
`
		fmt.Fprintf(os.Stderr, msg, what, pass)
		return 1
	} else if s != 0 {
		msg := `
Error running %s:
%s

`
		fmt.Fprintf(os.Stderr, msg, what, e)
		return 1
	}
	return 0
}

func run(args ...string) (string, string, int) {
	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if status, ok := e.Sys().(syscall.WaitStatus); ok {
				return stdout.String(), stderr.String(), status.ExitStatus()
			}
			log.Fatalf("Could not get an exit status from error: %v", e)
		}
		log.Fatalf("Error executing %s: %v", strings.Join(args, " "), err)
	}
	return stdout.String(), stderr.String(), 0
}
