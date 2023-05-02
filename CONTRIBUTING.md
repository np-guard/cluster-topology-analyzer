## Contributing In General
Our project welcomes external contributions. If you have an itch, please feel
free to scratch it.

To contribute code or documentation, please submit a [pull request](https://github.com/np-guard/cluster-topology-analyzer/pulls).

A good way to familiarize yourself with the codebase and contribution process is
to look for and tackle low-hanging fruit in the [issue tracker](https://github.com/np-guard/cluster-topology-analyzer/issues).

### Proposing new features

If you would like to implement a new feature, please [raise an issue](https://github.com/np-guard/cluster-topology-analyzer/issues)
before sending a pull request so the feature can be discussed. This is to avoid
you wasting your valuable time working on a feature that the project developers
are not interested in accepting into the code base.

### Fixing bugs

If you would like to fix a bug, please [raise an issue](https://github.com/np-guard/cluster-topology-analyzer/issues) before sending a
pull request so it can be tracked.

### Merge approval

The project maintainers will review any proposed code in a pull request. A change requires approval from at least one of the
maintainers.

For a list of the maintainers, see the [MAINTAINERS.md](MAINTAINERS.md) page.

## Legal

Each source file must include a license header for the Apache
Software License 2.0. Using the SPDX format is the simplest approach.
e.g.

```
/*
Copyright <holder> All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
```

We have tried to make it as easy as possible to make contributions. This
applies to how we handle the legal aspects of contribution. We use the
same approach - the [Developer's Certificate of Origin 1.1 (DCO)](https://github.com/hyperledger/fabric/blob/master/docs/source/DCO1.1.txt) - that the Linux® Kernel [community](https://elinux.org/Developer_Certificate_Of_Origin)
uses to manage code contributions.

We simply ask that when submitting a patch for review, the developer
must include a sign-off statement in the commit message.

Here is an example Signed-off-by line, which indicates that the
submitter accepts the DCO:

```
Signed-off-by: John Doe <john.doe@example.com>
```

You can include this automatically when you commit a change to your
local git repository using the following command:

```
git commit -s
```

## Communication
Please feel free to email each one of the [maintainers](MAINTAINERS.md).

## Setup
See [here](README.md#build-the-project).

## Testing
Test any change you make before submitting by running `make test` at the project's root directory.

## Coding style guidelines
We use [golangci-lint](https://golangci-lint.run/) to enforce style.
Make sure your changes are style compliant by running `make lint` at the project's root directory.

