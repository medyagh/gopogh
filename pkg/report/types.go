package report

// Version is gopogh version
var Version string

// Build includes commit sha date
var Build string

const (
	pass = "pass"
	fail = "fail"
	skip = "skip"
)

var resultTypes = [3]string{pass, fail, skip}
