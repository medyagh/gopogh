package report

// version is a private field and should be set when compiling with --ldflags="-X github.com/medyagh/gopogh/pkg/report.version=vX.Y.Z"
var version = "v0.0.0-unset"

// Build includes commit sha date
var Build string

const (
	pass = "pass"
	fail = "fail"
	skip = "skip"
)

var resultTypes = [3]string{pass, fail, skip}

// Version returns the version of gopogh
func Version() string {
	return version
}
