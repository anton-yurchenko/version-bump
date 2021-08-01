package langs

import (
	"fmt"

	changelog "github.com/anton-yurchenko/go-changelog"
)

var dockerRegex = []string{
	fmt.Sprintf("^LABEL .*[vV]ersion['\"]?=['\"]?[vV]?(?P<version>%v)['\"]?.*", changelog.SemVerRegex),
	fmt.Sprintf("^\\s+.*[vV]ersion['\"]?=['\"]?[vV]?(?P<version>%v)['\"]?.*", changelog.SemVerRegex),
}
