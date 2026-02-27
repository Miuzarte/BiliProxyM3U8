package templates

import (
	"embed"
	"fmt"
	"html"
	netUrl "net/url"
	"text/template"
)

//go:embed *.tmpl
var fs embed.FS

const MPD_TEMPLATE = `MPD.tmpl`

var MpdTemplate = template.Must(
	template.New(MPD_TEMPLATE).
		Funcs(template.FuncMap{
			"mul":        func(a, b int) int { return a * b },
			"add":        func(a, b int) int { return a + b },
			"htmlEscape": html.EscapeString,
			"urlEscape":  netUrl.QueryEscape,
			"formatDuration": func(seconds int) string {
				hours := seconds / 3600
				minutes := (seconds % 3600) / 60
				secs := seconds % 60
				if hours > 0 {
					return fmt.Sprintf("PT%dH%dM%dS", hours, minutes, secs)
				} else if minutes > 0 {
					return fmt.Sprintf("PT%dM%dS", minutes, secs)
				}
				return fmt.Sprintf("PT%dS", secs)
			},
		}).
		ParseFS(fs, MPD_TEMPLATE),
)

const M3U8_TEMPLATE = `M3U8.tmpl`

var M3u8Template = template.Must(
	template.New(M3U8_TEMPLATE).
		ParseFS(fs, M3U8_TEMPLATE),
)
