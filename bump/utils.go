package bump

import (
	"bufio"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
)

func getFiles(fs afero.Fs, dir string, excludeFiles []string) ([]string, error) {
	res := make([]string, 0)

	files, err := afero.ReadDir(fs, dir)
	if err != nil {
		return res, err
	}

main:
	for _, f := range files {
		if !f.IsDir() {
			for _, e := range excludeFiles {
				if path.Join(dir, f.Name()) == e {
					continue main
				}
			}
			res = append(res, f.Name())
		}
	}

	return res, nil
}

func filterFiles(configNames []string, files []string) []string {
	res := make([]string, 0)
	for _, f := range files {
		for _, n := range configNames {
			if strings.HasPrefix(n, "*.") {
				if strings.HasSuffix(f, strings.TrimPrefix(n, "*")) {
					res = append(res, f)
				}
			} else {
				if strings.HasSuffix(f, n) {
					res = append(res, f)
				}
			}
		}
	}

	return res
}

func readFile(fs afero.Fs, filepath string) ([]string, error) {
	lines := make([]string, 0)

	file, err := fs.Open(filepath)
	if err != nil {
		return lines, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, nil
}

func writeFile(fs afero.Fs, filepath string, content string) error {
	file, err := fs.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return errors.Wrap(err, "error opening a file")
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return errors.Wrap(err, "error writing to file")
	}

	if err := file.Sync(); err != nil {
		return errors.Wrap(err, "error saving changes to disk")
	}

	return nil
}
