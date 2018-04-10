# Installation

Tagged releases are [available from
GitHub](https://github.com/ActiveState/gometalinter-helper/releases).

You can use [`godownloader`](https://github.com/goreleaser/godownloader) to
generate a shell script that downloads and installs the latest release.

```
$> godownloader github.com/ActiveState/gometalinter-helper
```

You can also install the latest commit with:

```
$> go install github.com/ActiveState/gometalinter-helper/cmd/gometalinter-helper
```

## Usage

gometalinter-helper [-commit-hook] [-all] [-exe ...] -- [args to gometalinter]

This command wraps gometalinter to make it a bit simpler to use with a
commit hook or in a CI environment. You can also run it as a standalone
command.

It has a number of modes, depending on what arguments you pass it and the
presence of an environment variable named "CI".

If you pass the "-all" flag then it will check all Go files in the current
directory tree.

If you pass the "-commit-hook" flag then it will check new or modified files
that are about to be committed in a Git repo.

If the neither flag is passed and the CI environment variable is set, then it
will run a check of the current branch. If that branch is "master" than it
checks all Go files (like the "-all" flag). Otherwise it checks Go files in
the current branch that differ from master.

Finally, you can pass an explicit list of files to check. Note that if you
have files starting with a dash (-) this will probably blow up
horribly. Don't do that.

It will always ignore files in a directory named "vendor" or ".git".

It accepts the following arguments:

-all
  	Check all files in the tree.
-commit-hook
  	Check files that are staged for a commit.
-exe string
  	The name of the executable to run. (default "gometalinter")
-help
  	Show usage information.
-verbose
  	Be verbose about it.
