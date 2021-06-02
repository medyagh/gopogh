package templates

import (
	_ "embed"
)

//go:embed report3.css
var ReportCSS string

//go:embed report3.html
var ReportHTML string
