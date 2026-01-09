// Package templates contains the embedded CSS and HTML templates for the reports
package templates

import (
	_ "embed"
)

// ReportCSS is the CSS template for the report
//
//go:embed report3.css
var ReportCSS string

// ReportHTML is the HTML template for the report
//
//go:embed report3.html
var ReportHTML string
