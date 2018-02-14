<!-- MarkdownTOC -->

- [Contribution guidelines](#contribution-guidelines)
	- [Filing issues](#filing-issues)
	- [Contributing patches](#contributing-patches)

<!-- /MarkdownTOC -->

<a name="contribution-guidelines"></a>
# Contribution guidelines

<a name="filing-issues"></a>
## Filing issues

File issues using the standard
[Github issue tracker](https://github.com/fabric8-services/fabric8-jenkins-idler/) issues for this repository.
Before you submit a new issue, we recommend that you search the list of issues to see if anyone already submitted a similar issue.

<a name="contributing-patches"></a>
## Contributing patches

Thank you for your contributions! Please follow this process to submit a patch:

* Create an issue describing your proposed change to the repository.
* Fork the repository and create a topic branch.
  See also [Understanding the GitHub Flow](https://guides.github.com/introduction/flow/).
* Refer to the [README](./README.md) for how to build and test the _fabric8-jenkins-idler_.
* Submit a pull request with the proposed changes.
  To make reviewing and integration as smooth as possible, please run the `make all` target prior to submitting the pull request.
  Apart from compiling the source code and running the tests it will also make sure that the code adheres to coding standards and that the commit message matches the format `^Issue #[0-9]+  [A-Z]+.*`.
