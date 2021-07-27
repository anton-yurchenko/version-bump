package langs

const (
	Docker     string = "Docker"
	Go         string = "Go"
	JavaScript string = "JavaScript"
)

type Language struct {
	Name       string
	Files      []string
	Regex      *[]string
	JSONFields *[]string
}

func New(name string) *Language {
	switch name {
	case Docker:
		return &Language{
			Name:  Docker,
			Files: []string{"Dockerfile"},
			Regex: &dockerRegex,
		}
	case Go:
		return &Language{
			Name:  Go,
			Files: []string{"*.go"},
			Regex: &golangRegex,
		}
	case JavaScript:
		return &Language{
			Name: JavaScript,
			Files: []string{
				"package.json",
				"package-lock.json",
			},
			JSONFields: &javaScriptJSONFields,
		}
	default:
		return nil
	}
}
