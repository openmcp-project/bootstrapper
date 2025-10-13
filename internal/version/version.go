package version

import (
	"fmt"
	"runtime"
	"strings"

	apimachineryversion "k8s.io/apimachinery/pkg/version"
)

var (
	// These variables are set during build time using -ldflags
	// buildVersion is the semantic version of the build.
	buildVersion = ""
	// gitTreeState is either "clean" or "dirty" depending on the state of the git tree.
	gitTreeState = ""
	// gitCommit is the git commit hash of the build.
	gitCommit = ""
	// buildDate is the date of the build.
	buildDate = ""
)

// GetVersion returns the version information of the build.
func GetVersion() *apimachineryversion.Info {
	var (
		version  = strings.Split(buildVersion, ".")
		gitMajor string
		gitMinor string
	)

	if len(version) >= 2 {
		gitMajor = version[0]
		gitMinor = version[1]
	}

	return &apimachineryversion.Info{
		Major:        gitMajor,
		Minor:        gitMinor,
		GitVersion:   buildVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
