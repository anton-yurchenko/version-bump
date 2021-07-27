package langs

import (
	"fmt"

	changelog "github.com/anton-yurchenko/go-changelog"
)

var dockerRegex = []string{
	fmt.Sprintf("^LABEL\\s+\"[vV]ersion\"\\s*=\\s*\"[vV]?(?P<version>%v)\"", changelog.SemVerRegex),
}
