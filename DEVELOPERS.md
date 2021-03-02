# Developing Qri

* [Development Setup](#setup)
* [Coding Rules](#rules)
* [Commit Message Guidelines](#commits)
* [Troubleshooting](#troubleshooting)

## <a name="setup"></a> Development Setup

This document describes how to set up your development environment to build and test Qri, and
explains the basic mechanics of using `git`, `golint` and `go test`.

### Installing Dependencies

Before you can build Qri, you must install and configure the following dependencies on your
machine:

* [Git](http://git-scm.com/): The [Github Guide to
  Installing Git][git-setup] is a good source of information.

* [The Go Programming Language](https://golang.org): see golang.org to get started

* [golint](https://github.com/golang/lint): Golint is a linter for Go source code


### Forking Qri on Github

To contribute code to Qri, you must have a GitHub account so you can push code to your own
fork of Qri and open Pull Requests in the [GitHub Repository][github].

To create a Github account, follow the instructions [here](https://github.com/signup/free).
Afterwards, go ahead and [fork](http://help.github.com/forking) the
[Qri frontend repository][github].


### Building Qri


Check out this documentation on [how to build Qri from source](https://github.com/qri-io/qri/README.md#building-from-source)


## <a name="rules"></a> Coding Rules

When you push your branch to github and open up a pull request, it will automatically trigger  [CircleCI](https://circleci.com/about/) to lint and test your code.

In order to catch linting and testing errors before pushing the code to github, be sure to run `golint` and `go test`.

##### `golint`

Use `golint` to lint your code. Using `./...` indicates to `golint` that you want to lint each file in the current directory, and each file in each sub-directory you must be in the top level directory of the project in order to lint every file in the project:
```shell
$ golint ./...
```

No output indicates everything is styled correctly. Otherwise, the output will point you to which files/lines need to be changed in order to meet the go linting format.

##### `go test`

Use the built in `go test` command to test your code. Like the above, you can use `./...` to run each test file, if you are in the top most directory of the project:

```shell
$ go test ./...
?     github.com/qri-io/qri [no test files]
ok    github.com/qri-io/qri/actions 1.180s
ok    github.com/qri-io/qri/api 0.702s
ok    github.com/qri-io/qri/base  (cached)
ok    github.com/qri-io/qri/cmd 17.557s
?     github.com/qri-io/qri/cmd/generate  [no test files]
ok    github.com/qri-io/qri/config  (cached)
?     github.com/qri-io/qri/config/test [no test files]
?     github.com/qri-io/qri/docs  [no test files]
ok    github.com/qri-io/qri/lib 1.064s
ok    github.com/qri-io/qri/p2p (cached)
ok    github.com/qri-io/qri/p2p/test  (cached)
ok    github.com/qri-io/qri/repo  (cached)
ok    github.com/qri-io/qri/repo/fs (cached)
ok    github.com/qri-io/qri/profile  (cached)
ok    github.com/qri-io/qri/repo/test (cached)
ok    github.com/qri-io/qri/rev (cached)
```

Depending on what work you are doing and what has changed, tests may take up to a minute.

If everything is marked "ok", you are in the clear. Any extended output is a sign that a test has failed. Be sure to fix any bugs that are indicated or tests that no longer pass.

You may run just the tests for a specific package by running the `go test` command from that directory. You may also target specific tests to run based off a regex match of the test's name. For example, running `go test --run TestSetup` will execute any test that has the TestSetup prefix (tests named TestsSetupOne, TestSetup, and TestSetupWithAnotherName would all run). To see a verbose printout of all tests regardless of pass/skip/fail status, you may add the `-v` flag.

#### CLI Help Style

When creating or editing CLI commands, we try and follow a few style rules to keep things consistent:

- Make sure to mention all the positional arguments in the `Use` string. Choose descriptive names in all capital letters, and put optional arguments in square brackets.

- Always include a `Short` description. It should be a phrase, not a sentence: it starts with a lower-case letter and ends without a period.

- Long descriptions should be full sentences and start with a captial letter unless the first word is the name of a command or special word that should never be upper-case.

- Examples should be…
    - Indented with two spaces.
    - Example descriptions start with `#`.
    - Lines that should be typed start with `$`.

- Long text fields should not have leading or trailing blank lines.

- Try to use active, rather than passive, voice wherever possible.

- Use the `Args` field for limiting the number of arguments (or even better, for doing more detailed validation). Cobra has [some nice built-ins for this](https://github.com/spf13/cobra/#positional-and-custom-arguments).

Example:

```go
	cmd := &cobra.Command{
		Use:   "add DATASET [DATASET...]",
		Short: "add datasets from other peers",
		Long: `Add retrieves datasets owned by other peers and adds them to your repo. 
The reference names of the datasets will remain the same, including
the name of the peer that originally added the dataset. You must have 
` + "`qri connect`" + ` running in another terminal to use this command.`,
		Example: `  # Add a dataset named their_data, owned by other_peer:
  $ qri add other_peer/their_data`,
		Annotations: map[string]string{
			"group": "dataset",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(f); err != nil {
				return err
			}
			return o.Run(args)
		},
	}
```


## <a name="commits"></a> Git Commit Guidelines

We have very precise rules over how our git commit messages can be formatted.  This leads to **more
readable messages** that are easy to follow when looking through the **project history**.  But also,
we use the git commit messages to **generate the Qri change log**.

### Commit Message Format
Each commit message consists of a **header**, a **body** and a **footer**.  The header has a special
format that includes a **type**, a **scope** and a **subject**:

```
<type>(<scope>): <subject>
<BLANK LINE>
<body>
<BLANK LINE>
<footer>
```

The **header** is mandatory and the **scope** of the header is optional.

Any line of the commit message cannot be longer 100 characters! This allows the message to be easier
to read on GitHub as well as in various git tools.

### Revert
If the commit reverts a previous commit, it should begin with `revert: `, followed by the header
of the reverted commit.
In the body it should say: `This reverts commit <hash>.`, where the hash is the SHA of the commit
being reverted.
A commit with this format is automatically created by the [`git revert`][git-revert] command.

### Type
Must be one of the following:

* **feat**: A new feature
* **fix**: A bug fix
* **docs**: Documentation only changes
* **style**: Changes that do not affect the meaning of the code (white-space, formatting, missing
  semi-colons, etc)
* **refactor**: A code change that neither fixes a bug nor adds a feature
* **perf**: A code change that improves performance
* **test**: Adding missing or correcting existing tests
* **chore**: Changes to the build process or auxiliary tools and libraries such as documentation
  generation

### Scope
The scope could be anything specifying place of the commit change. For example, if I am refactoring something in the `api` package, I may start my commit with "refactor(api)". If it's something more specific, like the ListHandler, I may write "refactor(api/ListHandler)", or something similar. As long as it gets the point across on the scope of the refactor.

You can use `*` when the change affects more than a single scope.

### Subject
The subject contains succinct description of the change:

* use the imperative, present tense: "change" not "changed" nor "changes"
* don't capitalize first letter
* no dot (.) at the end

### Body
Just as in the **subject**, use the imperative, present tense: "change" not "changed" nor "changes".
The body should include the motivation for the change and contrast this with previous behavior.

### Footer
The footer should contain any information about **Breaking Changes** and is also the place to
[reference GitHub issues that this commit closes][closing-issues].

**Breaking Changes** should start with the word `BREAKING CHANGE:` with a space or two newlines.
The rest of the commit message is then used for this.

A detailed explanation can be found in this [document][commit-message-format].


[closing-issues]: https://help.github.com/articles/closing-issues-via-commit-messages/
[commit-message-format]: https://docs.google.com/document/d/1QrDFcIiPjSLDn3EL15IJygNPiHORgU1_OOAqWjiDU5Y/edit#
[git-revert]: https://git-scm.com/docs/git-revert
[git-setup]: https://help.github.com/articles/set-up-git
[github]: https://github.com/qri-io/frontend
[style]: https://standardjs.com
[yarn-install]: https://yarnpkg.com/en/docs/install


## <a name="troubleshooting"> Troubleshooting

#### "Too Many Open Files" on MacOS

If, while running the test suite, you get failing tests with "too many open files", you may need to adjust the number of files your shell process is willing to open. This is particularly common with the `p2p` and `cmd` pacakges. Both rely heavily on interacting temporary directories on the file system to run tests.

Often this is caused by having too low a file limit for your shell. You can use the [`ulimit` command](https://ss64.com/osx/ulimit.html) to check out or change the limits. Try the following command to set the shell limit to 1000 open files:

```
ulimit -S -n 1000
```

We recommend a ulimit value of at least 1000; feel free to go higher if you need. Some other software require values as high as 10000 to avoid the max open file issue.

###### This documentation has been adapted from the [Data Together](https://github.com/datatogether/datatogether), [Hyper](https://github.com/zeit/hyper), and [AngularJS](https://github.com/angular/angularJS) documentation, all of which are projects we :heart:
