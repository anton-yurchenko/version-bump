package console

import (
	"fmt"
	"os"
)

const (
	colorReset  string = "\033[0m"
	colorRed    string = "\033[31m"
	colorGreen  string = "\033[32m"
	colorYellow string = "\033[33m"
	colorCyan   string = "\033[36m"
)

func IncrementProjectVersion() {
	fmt.Println("Incrementing project version...")
}

func CommitingChanges() {
	fmt.Println("Commiting changes...")
}

func Language(name string) {
	fmt.Printf("  Updating %v%v%v files:\n",
		string(colorCyan),
		name,
		string(colorReset),
	)
}

func VersionUpdate(oldVersion, newVersion, filepath string) {
	fmt.Printf("    %v%v%v -> %v%v%v %v\n",
		string(colorYellow), oldVersion, string(colorReset),
		string(colorGreen), newVersion, string(colorReset),
		filepath,
	)
}

func UpdateAvailable(version string) {
	fmt.Printf("%vThe new version is available! Download from https://github.com/anton-yurchenko/version-bump/releases/tag/%v%v\n",
		string(colorGreen), version, string(colorReset),
	)
}

func ErrorCheckingForUpdate(msg interface{}) {
	fmt.Printf("%vError checking for update: %v%v\n",
		string(colorYellow), msg, string(colorReset),
	)
}

func Error(msg interface{}) {
	fmt.Printf("%v%v%v\n",
		string(colorRed), msg, string(colorReset),
	)
}

func Fatal(msg interface{}) {
	Error(msg)
	os.Exit(1)
}
