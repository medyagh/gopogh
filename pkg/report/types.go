package report

// Version is gopogh version
var Version = "v0.0.15"

// Build includes commit sha date
var Build string

const (
	pass = "pass"
	fail = "fail"
	skip = "skip"
)

var resultTypes = [3]string{pass, fail, skip}
