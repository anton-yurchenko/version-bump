# version-bump

[![Release](https://img.shields.io/github/v/release/anton-yurchenko/version-bump)](https://github.com/anton-yurchenko/version-bump/releases/latest)
[![Code Coverage](https://codecov.io/gh/anton-yurchenko/version-bump/branch/main/graph/badge.svg)](https://codecov.io/gh/anton-yurchenko/version-bump)
[![Go Report Card](https://goreportcard.com/badge/github.com/anton-yurchenko/version-bump)](https://goreportcard.com/report/github.com/anton-yurchenko/version-bump)
[![Release](https://github.com/anton-yurchenko/version-bump/actions/workflows/release.yml/badge.svg)](https://github.com/anton-yurchenko/version-bump/actions/workflows/release.yml)
[![License](https://img.shields.io/github/license/anton-yurchenko/version-bump)](LICENSE.md)

Have you ever made a mistake incrementing a project version?  
Do you have multiple files to update the version at?  
I was always forgetting to update a `Dockerfile` label or a version constant in `main.go` file. Inspired by `npm bump`, I wrote **version-bump**.  

This application allows easily incrementing a single/multi language project [Semantic Version](https://semver.org/), committing the changes and tagging a commit.

![PIC](docs/images/demo.png)

## Features

- Supported languages: **Go**, **Docker**, **JavaScript**
- [Semantic Versioning](https://semver.org/) Compliant
- Update files in multiple directories of the project at once
- Commit and tag changes

## Installation

Download [latest release](https://github.com/anton-yurchenko/version-bump/releases/latest) and move the file to one of the directories under your `PATH` environmental variable.

## Configuration

**version-bump** has two modes of operation: automatic / manual.
In automatic mode, **version-bump** will try to identify versions of all supported languages in a root of the project (wherever executed).
In a manual mode, **version-bump** will read a configuration file and modify files according to it. It is expected be executed in a root of the project where the configuration file is.

Some languages, have a constant value in a specific file that contains a version, which are fairly easy to increment.
But some languages are leaving that decision to a developer, thus **version-bump** assumes a constant position/value for them as well.

| Language      | Expected Values                               | Filename                              |
|:-------------:|:---------------------------------------------:|:-------------------------------------:|
| Docker        | `org.opencontainers.image.version` label      | `Dockerfile`                          |
| Go            | String constant named `Version`/`version`     | `*.go`                                |
| JavaScript    | JSON `version` field                          | `package.json`, `package-lock.json`   |

### Automatic

Run **version-bump** in a root of the project: `version-bump <major/minor/patch>`

### Manual

1. Create a configuration `.bump` file in a root of the project.
2. Add project languages and their configuration in a form of:

    ```toml
    [ <language-name> ]
    enabled = true/false
    directories = [ <path>, <path>, ... ]
    exclude_files = [ <path>, <path>, ... ]
    ```

    - `<language-name>` - one of `[ 'docker', 'go', 'javascript' ]`
    - `enabled` - default `false`
    - `directories` - default `['.']`
    - `exclude_files` - default `[]`

3. Run **version-bump** in a root of the project: `version-bump <major/minor/patch>`

*Configuration Example:*

```toml
[docker]
enabled = true
directories = [ '.', 'tools/qa' ]

[go]
enabled = true
directories = [ 'server', 'tools/cli', 'tools/qa' ]
exclude_files = [ 'server/server_test.go', 'tools/qa/main_test.go' ]

[javascript]
enabled = true
directories = [ 'client' ]
```

## Remarks

- Versions are expected to be consistent across all files
- In automatic mode, **version-bump** has all languages enabled

## License

[MIT](LICENSE.md) © 2021-present Anton Yurchenko
