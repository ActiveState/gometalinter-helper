This command can be used to check Go file for conformance with various
tools. It is intended to be used both as part of your Git commit hooks and
as a tool to be run under CI. You can also run it as a standalone command.

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
