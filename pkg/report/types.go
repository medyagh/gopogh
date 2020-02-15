package report

// Version is gopogh version
const Version = "v0.1.1"

// Build includes commit sha date
var Build string

const (
	pass = "pass"
	fail = "fail"
	skip = "skip"
)

var resultTypes = [3]string{pass, fail, skip}
