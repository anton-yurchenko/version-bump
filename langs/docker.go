package langs

import (
	"fmt"

	changelog "github.com/anton-yurchenko/go-changelog"
)

var dockerRegex = []string{
	fmt.Sprintf("^LABEL .*org.opencontainers.image.version['\"= ]*[vV]?(?P<version>%v)['\"]?.*", changelog.SemVerRegex),
	fmt.Sprintf("^\\s*['\"]?org.opencontainers.image.version['\"= ]*[vV]?(?P<version>%v)['\"]?.*", changelog.SemVerRegex),
}
