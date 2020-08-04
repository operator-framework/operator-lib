# How to contribute

operator-lib is Apache 2.0 licensed and accepts contributions via
GitHub pull requests. This document outlines some of the conventions
on commit message formatting, contact points for developers, and
other resources to help get contributions into operator-lib.

# Email and Chat

- Email: [operator-framework][operator_framework]
- Slack: [#operator-sdk-dev][operator-sdk-dev]

## Getting started

- Fork the repository on GitHub
- Test your code using `make test`
- Check the linter with `make lint`
- For more information what targets are available run `make help`

## Reporting bugs and creating issues

Reporting bugs is one of the best ways to contribute. If any part of the
operator-lib project has bugs, please let us know by
[opening an issue][reporting-issues].

To make the bug report accurate and easy to understand, please try to create bug
reports that are:

* Specific. Include as much details as possible: which version, what
  environment, what configuration, etc.
* Reproducible. Include the steps to reproduce the problem.
* Unique. Do not duplicate existing bug reports.
* Scoped. One bug per report. Do not follow up with another bug inside one
  report.

## Contribution flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where to base the contribution. This is usually
  `main`.
- Make commits of logical units.
- Make sure commit messages are in the proper format (see below).
- Push changes in a topic branch to a personal fork of the repository.
- Submit a pull request to operator-framework/operator-lib.
- The PR must receive a LGTM from two maintainers found in the OWNERS file.

Thanks for contributing!

### Code style

The coding style suggested by the Golang community is used in operator-lib.
See the [style doc][golang-style-doc] for details.

Please follow this style to make operator-lib easy to review, maintain and develop.

### Format of the commit message

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why.

```
add the test-cluster command

this uses tmux to setup a test cluster that can easily be killed and started for debugging.

Fixes #38
```

The format can be described more formally as follows:

```
<what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

The first line is the subject and should be no longer than 70 characters, the second line is always blank, and other lines should be wrapped at 80 characters. This allows the message to be easier to read on GitHub as well as in various git tools.

## Documentation

If the contribution changes the existing APIs or user interface it must include sufficient documentation to explain the use of the new or updated feature.

[operator_framework]: https://groups.google.com/forum/#!forum/operator-framework
[reporting-issues]: https://github.com/operator-framework/operator-lib/issues/new
[golang-style-doc]: https://github.com/golang/go/wiki/CodeReviewComments
[operator-sdk-dev]: https://kubernetes.slack.com/archives/C017UU45SHL
