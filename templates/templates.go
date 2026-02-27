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

type MpdData struct {
	Title         string
	OwnerName     string
	Aid           int
	Bvid          string
	TotalDuration int
	Periods       []PeriodData
}

type PeriodData struct {
	Duration        int
	VideoURL        string
	VideoMimeType   string
	VideoCodecs     string
	VideoBandwidth  int
	VideoWidth      int
	VideoHeight     int
	VideoFrameRate  string
	VideoInitRange  string
	VideoIndexRange string
	AudioURL        string
	AudioMimeType   string
	AudioCodecs     string
	AudioBandwidth  int
	AudioInitRange  string
	AudioIndexRange string
}

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

type M3u8Data struct {
	Title string
	Items []M3u8Item
}

type M3u8Item struct {
	Duration int
	Title    string
	URL      string
}

const M3U8_TEMPLATE = `M3U8.tmpl`

var M3u8Template = template.Must(
	template.New(M3U8_TEMPLATE).
		ParseFS(fs, M3U8_TEMPLATE),
)
