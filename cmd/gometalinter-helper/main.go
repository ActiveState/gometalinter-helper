package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"

	zglob "github.com/mattn/go-zglob"
)

type helper struct {
	hook     bool
	all      bool
	exe      string
	verbose  bool
	ignore   []string
	dirs     []string
	fileArgs []string
	gmlArgs  []string
}

type stringFlags []string

func (f *stringFlags) String() string {
	s := ""
	for _, v := range *f {
		s += fmt.Sprintf("-ignore %s", v)
	}
	return s
}

func (f *stringFlags) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func main() {
	h := &helper{}

	flag.BoolVar(&h.hook, "commit-hook", false, "Check files that are staged for a commit.")
	flag.BoolVar(&h.all, "all", false, "Check all files in the tree.")
	flag.StringVar(&h.exe, "exe", "gometalinter", "The name of the executable to run.")
	var ignore stringFlags
	flag.Var(&ignore, "ignore", "Ignore all files listed in the given file. The file should be in .gitignore format. Can be passed multiple times.")
	flag.BoolVar(&h.verbose, "verbose", false, "Be verbose about it.")
	var help bool
	flag.BoolVar(&help, "help", false, "Show usage information.")
	flag.Parse()

	if help {
		usage("")
		os.Exit(0)
	}

	if len(ignore) != 0 {
		h.readIgnoreFiles(ignore)
	}

	h.parseExtraArgs()
	h.setDirs()
	h.checkArgSanity()

	if len(h.dirs) == 0 {
		os.Exit(0)
	}

	os.Exit(h.runMetalinter())
}

func usage(err string) {
	if err != "" {
		fmt.Printf("\n  *** %s ***\n", err)
	}

	fmt.Print(`
 gometalinter-helper [-commit-hook] [-all] [-exe ...] -- [args to gometalinter]

  This command wraps gometalinter to make it a bit simpler to use with a
  commit hook or in a CI environment. You can also run it as a standalone
  command.

  It has a number of modes, depending on what arguments you pass it and the
  presence of an environment variable named "CI".

  * -all - If you pass this flag then it will check all Go files in the
    current directory tree.

  * -commit-hook - If you pass this flag then it will check new or modified
    files that are about to be committed in a Git repo.

  * -ignore - If you have files with zglob ignore patterns like .gitignore you
    can pass these files via the "-ignore". Any files matching these patterns
    will be ignored.

  If the neither "-all" nor "-commit-hook" is passed and the "CI" environment
  variable is set, then it will run a check of the current branch. If that
  branch is "master" than it checks all Go files (like the "-all"
  flag). Otherwise it checks Go files in the current branch that differ from
  master.

  Finally, you can pass an explicit list of files to check. Note that if you
  have files starting with a dash (-) this will probably blow up
  horribly. Don't do that.

  It will always ignore files in a directory named "vendor" or ".git".

  It accepts the following arguments:

`)
	flag.PrintDefaults()
}

func (h *helper) readIgnoreFiles(files []string) {
	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			fmt.Printf("Could not open %s for reading: %s", file, err)
			os.Exit(1)
		}
		// nolint: errcheck
		defer f.Close()

		s := bufio.NewScanner(f)
		for s.Scan() {
			if regexp.MustCompile(`\S`).MatchString(s.Text()) {
				h.ignore = append(h.ignore, strings.TrimSpace(s.Text()))
			}
		}
	}
}

func (h *helper) parseExtraArgs() {
	inGMLArgs := false
	for _, v := range flag.Args() {
		if v == "--" {
			inGMLArgs = true
			continue
		}
		// If this command is passed additional dashless args (file names) and
		// _then_ some GML args after a "--", then we'll see the "--" in the
		// values returned by flag.Args(). However, if there are no files,
		// just "--" followed by GML args, then the "--" is hidden by the
		// flags package so we have to rely on the fact that anything starting
		// with a dash _should_ be a GML arg.
		if inGMLArgs || regexp.MustCompile(`^-`).MatchString(v) {
			h.gmlArgs = append(h.gmlArgs, v)
		} else {
			h.fileArgs = append(h.fileArgs, v)
		}
	}
}

func (h *helper) checkArgSanity() {
	if len(h.fileArgs) > 0 && (h.all || h.hook) {
		badUsage("Cannot combine -all or -commit-hook with files")
	}

	if len(h.fileArgs) == 0 && !(h.all || h.hook) && !inCI() {
		badUsage("When running outside a CI system, you must pass -all, -commit-hook, or a list of files")
	}

	if h.all && h.hook {
		badUsage("Cannot pass both -all and -commit-hook")
	}
}

func badUsage(e string) {
	usage(e)
	os.Exit(1)
}

func (h *helper) setDirs() {
	var files []string
	if h.all {
		files = h.allGoFiles()
	} else if h.hook {
		files = h.commitHookGoFiles()
	} else if inCI() {
		files = h.ciGoFiles()
	} else {
		files = h.fileArgs
	}

	dirs := make(map[string]bool)
Files:
	for _, f := range files {
		for _, i := range h.ignore {
			m, err := zglob.Match(i, f)
			if err != nil {
				fmt.Printf("Error with zglob (%s): %s", i, err)
			}
			if m {
				continue Files
			}
		}

		info, err := os.Stat(f)
		if err != nil {
			fmt.Printf("Error stat'ing %s\n", f)
			os.Exit(1)
		}
		if info.IsDir() {
			dirs[f] = true
		} else {
			dirs[filepath.Dir(f)] = true
		}
	}

	for d := range dirs {
		h.dirs = append(h.dirs, d)
	}

	sort.Strings(h.dirs)
}

func inCI() bool {
	return os.Getenv("CI") != ""
}

func (h *helper) allGoFiles() []string {
	if h.verbose {
		fmt.Println("Will check all go files under the current directory")
	}

	var files []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && (path == ".git" || path == "vendor") {
			return filepath.SkipDir
		}

		if isGoFile(path) {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		fmt.Printf("Error walking tree: %v\n", err)
		os.Exit(1)
	}

	return files
}

func (h *helper) commitHookGoFiles() []string {
	if h.verbose {
		fmt.Println("Will check go files staged for commit")
	}

	return filteredOutput(stdoutFrom("git", "diff", "--cached", "--name-only", "--diff-filter=ACM"))
}

func (h *helper) ciGoFiles() []string {
	branch := strings.TrimSpace(stdoutFrom("git", "rev-parse", "--abbrev-ref", "HEAD"))
	if branch == "master" {
		if h.verbose {
			fmt.Println("In CI mode for master branch")
		}
		return h.allGoFiles()
	}

	if h.verbose {
		fmt.Println("In CI mode for non-master branch")
		fmt.Println("Will check all go files that have changed in this branch")
	}
	return filteredOutput(stdoutFrom("git", "diff", "--name-only", "master..."+branch))
}

func stdoutFrom(args ...string) string {
	cmd := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running %s: %v\nStderr:\n%s\n", strings.Join(args, " "), err, stderr.String())
		os.Exit(1)
	}
	return stdout.String()
}

func isGoFile(path string) bool {
	return regexp.MustCompile(`\.go$`).MatchString(path)
}

func filteredOutput(output string) []string {
	var files []string
	for _, l := range linesFrom(strings.TrimSpace(output)) {
		p := strings.Split(filepath.ToSlash(l), "/")
		if p[0] == ".git" || p[0] == "vendor" {
			continue
		}
		files = append(files, l)
	}
	return files
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

func (h *helper) runMetalinter() int {
	args := h.gmlArgs
	args = append(args, h.dirs...)
	if h.verbose {
		fmt.Printf("Executing %s %s\n", h.exe, strings.Join(args, " "))
	}
	cmd := exec.Command(h.exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if status, ok := e.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}
		fmt.Printf("Error executing [%s %s]: %v\n", h.exe, strings.Join(args, " "), err)
		return 1
	}
	return 0
}
