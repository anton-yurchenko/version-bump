package bump_test

import (
	"fmt"
	"path"
	"testing"
	"version-bump/bump"
	"version-bump/mocks"

	"github.com/pkg/errors"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNew(t *testing.T) {
	a := assert.New(t)

	type configFile struct {
		Exists  bool
		Content string
	}

	type test struct {
		ConfigFile            configFile
		ExpectedConfiguration bump.Configuration
		ExpectedError         string
	}

	suite := map[string]test{
		"Automatic": {
			ConfigFile: configFile{},
			ExpectedConfiguration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
				Go: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
				JavaScript: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			ExpectedError: "",
		},
		"Docker": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[docker]
enabled = true
directories = ['dir1','dir2']`,
			},
			ExpectedConfiguration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"dir1", "dir2"},
				},
				Go: bump.Language{
					Enabled:     false,
					Directories: []string{"."},
				},
				JavaScript: bump.Language{
					Enabled:     false,
					Directories: []string{"."},
				},
			},
			ExpectedError: "",
		},
		"Go": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[go]
enabled = true
directories = ['dir1','dir2']`,
			},
			ExpectedConfiguration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     false,
					Directories: []string{"."},
				},
				Go: bump.Language{
					Enabled:     true,
					Directories: []string{"dir1", "dir2"},
				},
				JavaScript: bump.Language{
					Enabled:     false,
					Directories: []string{"."},
				},
			},
			ExpectedError: "",
		},
		"JavaScript": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[javascript]
enabled = true
directories = ['dir1','dir2']`,
			},
			ExpectedConfiguration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     false,
					Directories: []string{"."},
				},
				Go: bump.Language{
					Enabled:     false,
					Directories: []string{"."},
				},
				JavaScript: bump.Language{
					Enabled:     true,
					Directories: []string{"dir1", "dir2"},
				},
			},
			ExpectedError: "",
		},
		"Complex": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[docker]
enabled = true
directories = [ '.', 'tools/qa' ]
				
[go]
enabled = true
directories = [ 'server', 'tools/cli', 'tools/qa' ]
				
[javascript]
enabled = true
directories = [ 'client' ]`,
			},
			ExpectedConfiguration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{".", "tools/qa"},
				},
				Go: bump.Language{
					Enabled:     true,
					Directories: []string{"server", "tools/cli", "tools/qa"},
				},
				JavaScript: bump.Language{
					Enabled:     true,
					Directories: []string{"client"},
				},
			},
			ExpectedError: "",
		},
		"Exclude Files": {
			ConfigFile: configFile{
				Exists: true,
				Content: `[docker]
enabled = true
directories = [ '.', 'tools/qa' ]
exclude_files = [ 'tools/qa/Dockerfile' ]
				
[go]
enabled = true
directories = [ 'server', 'tools/cli', 'tools/qa' ]
exclude_files = [ 'tools/cli/main_test.go' ]
				
[javascript]
enabled = true
directories = [ 'client' ]
exclude_files = [ 'client/test.js' ]`,
			},
			ExpectedConfiguration: bump.Configuration{
				Docker: bump.Language{
					Enabled:      true,
					Directories:  []string{".", "tools/qa"},
					ExcludeFiles: []string{"tools/qa/Dockerfile"},
				},
				Go: bump.Language{
					Enabled:      true,
					Directories:  []string{"server", "tools/cli", "tools/qa"},
					ExcludeFiles: []string{"tools/cli/main_test.go"},
				},
				JavaScript: bump.Language{
					Enabled:      true,
					Directories:  []string{"client"},
					ExcludeFiles: []string{"client/test.js"},
				},
			},
			ExpectedError: "",
		},
	}

	var counter int
	for name, test := range suite {
		counter++
		t.Logf("Test Case %v/%v - %s", counter, len(suite), name)

		fs := afero.NewMemMapFs()
		meta := memfs.New()
		data := memfs.New()

		_, err := git.Init(
			filesystem.NewStorage(meta, cache.NewObjectLRU(cache.DefaultMaxSize)),
			data,
		)
		if err != nil {
			t.Errorf("error preparing test case: error initializing repository: %v", err)
			continue
		}

		if test.ConfigFile.Exists {
			f, err := fs.Create(".bump")
			if err != nil {
				t.Errorf("error preparing test case: error creating Docker files: %v", err)
				continue
			}

			_, err = f.WriteString(test.ConfigFile.Content)
			if err != nil {
				t.Errorf("error preparing test case: error writing Docker files: %v", err)
				continue
			}
		}

		b, err := bump.New(fs, meta, data, ".")
		if test.ExpectedError != "" || err != nil {
			a.EqualError(err, test.ExpectedError)
			a.Equal(nil, b)
		} else {
			a.Equal(fs, b.FS)
			a.Equal(test.ExpectedConfiguration, b.Configuration)
			a.NotEqual(nil, b.Git)
		}
	}
}

func TestBump(t *testing.T) {
	a := assert.New(t)

	type file struct {
		Name                string
		ExpectedToBeChanged bool
		Content             string
	}

	type allFiles struct {
		Docker     map[string][]file
		Go         map[string][]file
		JavaScript map[string][]file
	}

	type test struct {
		Version            string
		Configuration      bump.Configuration
		Files              allFiles
		Action             int
		MockAddError       error
		MockCommitError    error
		MockCreateTagError error
		ExpectedError      string
	}

	suite := map[string]test{
		"Empty Configuration": {
			Version:            "",
			Configuration:      bump.Configuration{},
			Files:              allFiles{},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "0 files updated",
		},
		"Docker - Single, Lowercase, without Quotes, without Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "version"="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Single, Lowercase, with Quotes, without Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL version="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Single, Uppercase, with Quotes, with Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "Version"="v1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Single, Uppercase, without Quotes, with Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL Version="v1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multiple, Lowercase, with Quotes, without Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "repository"="https://github.com/anton-yurchenko/git-release" "version"="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multiple, Lowercase, without Quotes, without Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "repository"="https://github.com/anton-yurchenko/git-release" version="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multiple, Uppercase, with Quotes, with Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "repository"="https://github.com/anton-yurchenko/git-release" "Version"="v1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multiple, Uppercase, without Quotes, with Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "repository"="https://github.com/anton-yurchenko/git-release" Version="v1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multi-line, Lowercase, with Quotes, without Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release" \
	"version"="1.2.3" \
	"maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multi-line, Lowercase, without Quotes, without Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release" \
	version="1.2.3" \
	"maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multi-line, Uppercase, with Quotes, with Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release" \
	"Version"="v1.2.3" \
	"maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Multi-line, Uppercase, without Quotes, with Prefix": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release" \
	Version="v1.2.3" \
	"maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Go - Single Constant": {
			Version: "1.3.0",
			Configuration: bump.Configuration{
				Go: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"

const Version string = "1.2.3"

func main() {
	fmt.Println(Version)
}`,
						},
					},
				},
			},
			Action:             bump.Minor,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Go - Single Constant #2": {
			Version: "1.2.4",
			Configuration: bump.Configuration{
				Go: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"

const Version := "1.2.3"

func main() {
	fmt.Println(Version)
}`,
						},
					},
				},
			},
			Action:             bump.Patch,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Go - Multiple Constants": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Go: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"

const (
	Version                                          string = "1.2.4"
	SomeVeryLongVariableNameThatAddsALotOfWhitespace string = "abc"
)

func main() {
	fmt.Println(Version)
}`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"JavaScript - Multiple Constants": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				JavaScript: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				JavaScript: map[string][]file{
					".": {
						{
							Name:                "package.json",
							ExpectedToBeChanged: true,
							Content: `{
	"name": "git-release",
	"version": "1.2.3",
	"description": "A GitHub Action for creating a GitHub Release with Assets and Changelog whenever a new Tag is pushed to the repository.",
	"main": "wrapper.js",
	"directories": {
	  "doc": "docs"
	},
	"repository": {
	  "type": "git",
	  "url": "git+https://github.com/anton-yurchenko/git-release.git"
	},
	"keywords": [],
	"author": "Anton Yurchenko",
	"license": "MIT",
	"bugs": {
	  "url": "https://github.com/anton-yurchenko/git-release/issues"
	},
	"homepage": "https://github.com/anton-yurchenko/git-release#readme",
	"dependencies": {
	  "@actions/core": "^1.4.0"
	},
	"devDependencies": {}
}`,
						},
						{
							Name:                "package-lock.json",
							ExpectedToBeChanged: true,
							Content: `{
	"name": "git-release",
	"version": "1.2.3",
	"lockfileVersion": 2,
	"requires": true,
	"packages": {
	  "": {
		"version": "1.2.3",
		"license": "MIT",
		"dependencies": {
		  "@actions/core": "^1.4.0"
		},
		"devDependencies": {}
	  },
	  "node_modules/@actions/core": {
		"version": "1.4.0",
		"resolved": "https://registry.npmjs.org/@actions/core/-/core-1.4.0.tgz",
		"integrity": "sha512-CGx2ilGq5i7zSLgiiGUtBCxhRRxibJYU6Fim0Q1Wg2aQL2LTnF27zbqZOrxfvFQ55eSBW0L8uVStgtKMpa0Qlg=="
	  }
	},
	"dependencies": {
	  "@actions/core": {
		"version": "1.4.0",
		"resolved": "https://registry.npmjs.org/@actions/core/-/core-1.4.0.tgz",
		"integrity": "sha512-CGx2ilGq5i7zSLgiiGUtBCxhRRxibJYU6Fim0Q1Wg2aQL2LTnF27zbqZOrxfvFQ55eSBW0L8uVStgtKMpa0Qlg=="
	  }
	}
}`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
		"Docker - Get Files Error": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"dir"},
				},
			},
			Files:              allFiles{},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "error incrementing version in Docker project: error listing directory files: open dir: file does not exist",
		},
		"Go - Get Files Error": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Go: bump.Language{
					Enabled:     true,
					Directories: []string{"dir"},
				},
			},
			Files:              allFiles{},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "error incrementing version in Go project: error listing directory files: open dir: file does not exist",
		},
		"JavaScript - Get Files Error": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				JavaScript: bump.Language{
					Enabled:     true,
					Directories: []string{"dir"},
				},
			},
			Files:              allFiles{},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "error incrementing version in JavaScript project: error listing directory files: open dir: file does not exist",
		},
		"Inconsistent Versioning": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
				Go: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "version"="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"

const Version string = "1.3.0"

func main() {
	fmt.Println(Version)
}`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "inconsistent versioning",
		},
		"Save Error": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "version"="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    errors.New("reason"),
			MockCreateTagError: nil,
			ExpectedError:      "error commiting changes: error commiting changes: reason",
		},
		"Exclude Files": {
			Version: "2.0.0",
			Configuration: bump.Configuration{
				Docker: bump.Language{
					Enabled:     true,
					Directories: []string{"."},
				},
				Go: bump.Language{
					Enabled:      true,
					Directories:  []string{".", "lib"},
					ExcludeFiles: []string{"lib/lib_test.go"},
				},
			},
			Files: allFiles{
				Docker: map[string][]file{
					".": {
						{
							Name:                "Dockerfile",
							ExpectedToBeChanged: true,
							Content: `FROM golang:1.16 as builder
WORKDIR /opt/src
COPY . .
RUN groupadd -g 1000 appuser &&\
	useradd -m -u 1000 -g appuser appuser

RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o /opt/app
FROM scratch
LABEL "repository"="https://github.com/anton-yurchenko/git-release"
LABEL "maintainer"="Anton Yurchenko <anton.doar@gmail.com>"
LABEL "version"="1.2.3"
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY LICENSE.md /LICENSE.md
COPY --from=builder --chown=1000:0 /opt/app /app
ENTRYPOINT [ "/app" ]`,
						},
					},
				},
				Go: map[string][]file{
					".": {
						{
							Name:                "main.go",
							ExpectedToBeChanged: true,
							Content: `package main

import "fmt"
							
const Version string = "1.2.3"
							
func main() {
	fmt.Println(Version)
}`,
						},
					},
					"lib": {
						{
							Name:                "lib.go",
							ExpectedToBeChanged: true,
							Content: `package lib

import "fmt"

const Version string = "1.2.3"`,
						},
						{
							Name:                "lib_test.go",
							ExpectedToBeChanged: false,
							Content: `package lib_test

import "fmt"

const Version string = "1.2.3"`,
						},
					},
				},
			},
			Action:             bump.Major,
			MockAddError:       nil,
			MockCommitError:    nil,
			MockCreateTagError: nil,
			ExpectedError:      "",
		},
	}

	var counter int
	for name, test := range suite {
		counter++
		t.Logf("Test Case %v/%v - %s", counter, len(suite), name)

		m1 := new(mocks.Repository)
		m2 := new(mocks.Worktree)

		r := bump.Bump{
			FS: afero.NewMemMapFs(),
			Git: bump.GitConfig{
				UserName:   username,
				UserEmail:  email,
				Repository: m1,
				Worktree:   m2,
			},
			Configuration: test.Configuration,
		}

		shouldBeCommitted := false

		if test.Configuration.Docker.Enabled {
			for _, dir := range test.Configuration.Docker.Directories {
				for tgtDir, tgtFiles := range test.Files.Docker {
					if dir == tgtDir {
						for _, tgtFile := range tgtFiles {
							shouldBeCommitted = true
							f, err := r.FS.Create(path.Join(dir, tgtFile.Name))
							if err != nil {
								t.Errorf("error preparing test case: error creating Docker files: %v", err)
								continue
							}

							_, err = f.WriteString(tgtFile.Content)
							if err != nil {
								t.Errorf("error preparing test case: error writing Docker files: %v", err)
								continue
							}
						}
					}
				}
			}
		}

		if test.Configuration.Go.Enabled {
			for _, dir := range test.Configuration.Go.Directories {
				for tgtDir, tgtFiles := range test.Files.Go {
					if dir == tgtDir {
						for _, tgtFile := range tgtFiles {
							shouldBeCommitted = true
							f, err := r.FS.Create(path.Join(dir, tgtFile.Name))
							if err != nil {
								t.Errorf("error preparing test case: error creating Go files: %v", err)
								continue
							}

							_, err = f.WriteString(tgtFile.Content)
							if err != nil {
								t.Errorf("error preparing test case: error writing Go files: %v", err)
								continue
							}
						}
					}
				}
			}
		}

		if test.Configuration.JavaScript.Enabled {
			for _, dir := range test.Configuration.JavaScript.Directories {
				for tgtDir, tgtFiles := range test.Files.JavaScript {
					if dir == tgtDir {
						for _, tgtFile := range tgtFiles {
							shouldBeCommitted = true
							f, err := r.FS.Create(path.Join(dir, tgtFile.Name))
							if err != nil {
								t.Errorf("error preparing test case: error creating JavaScript files: %v", err)
								continue
							}

							_, err = f.WriteString(tgtFile.Content)
							if err != nil {
								t.Errorf("error preparing test case: error writing JavaScript files: %v", err)
								continue
							}
						}
					}
				}
			}
		}

		if shouldBeCommitted {
			for dir, files := range test.Files.Docker {
				for _, file := range files {
					if file.ExpectedToBeChanged {
						var f string
						if dir == "." {
							f = file.Name
						} else {
							f = path.Join(dir, file.Name)
						}
						m2.On("Add", f).Return(nil, test.MockAddError).Once()
					}
				}
			}

			for dir, files := range test.Files.Go {
				for _, file := range files {
					if file.ExpectedToBeChanged {
						var f string
						if dir == "." {
							f = file.Name
						} else {
							f = path.Join(dir, file.Name)
						}
						m2.On("Add", f).Return(nil, test.MockAddError).Once()
					}
				}
			}

			for dir, files := range test.Files.JavaScript {
				for _, file := range files {
					if file.ExpectedToBeChanged {
						var f string
						if dir == "." {
							f = file.Name
						} else {
							f = path.Join(dir, file.Name)
						}
						m2.On("Add", f).Return(nil, test.MockAddError).Once()
					}
				}
			}

			hash := plumbing.NewHash("abc")

			m2.On(
				"Commit", test.Version, mock.AnythingOfType("*git.CommitOptions"),
			).Return(hash, test.MockCommitError).Once()

			m1.On(
				"CreateTag", fmt.Sprintf("v%v", test.Version), hash, mock.AnythingOfType("*git.CreateTagOptions"),
			).Return(nil, test.MockCreateTagError).Once()
		}

		err := r.Bump(test.Action)
		if test.ExpectedError != "" || err != nil {
			a.EqualError(err, test.ExpectedError)
		}
	}
}
