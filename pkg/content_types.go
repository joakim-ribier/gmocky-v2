package pkg

import (
	"strings"

	"github.com/joakim-ribier/go-utils/pkg/slicesutil"
)

var CONTENT_TYPES = []string{
	"application/json",
	"application/x-www-form-urlencoded",
	"application/xhtml+xml",
	"application/xml",
	"image/jpeg",
	"image/png",
	"image/svg+xml",
	"multipart/form-data",
	"text/css",
	"text/csv",
	"text/html",
	"text/json",
	"text/plain",
	"text/xml",
}

var IS_DISPLAY_CONTENT = slicesutil.FilterT(CONTENT_TYPES, func(arg string) bool {
	return arg == "application/json" || arg == "application/xml" || strings.Contains(arg, "text/")
})

var CHARSET = []string{
	"UTF-8",
	"ISO-8859-1",
	"UTF-16",
}
