package main

import (
	"fmt"
	"html"
	"net/http"
	netUrl "net/url"
	"strconv"
	"text/template"

	"github.com/Miuzarte/biligo"
	"github.com/rs/zerolog/log"
)

const MPD_TEMPLATE = `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="urn:mpeg:dash:schema:mpd:2011 DASH-MPD.xsd"
    type="static"
    minBufferTime="PT1.5S"
    mediaPresentationDuration="{{.TotalDuration | formatDuration}}"
    profiles="urn:mpeg:dash:profile:isoff-main:2011">

    <ProgramInformation>
        <Title>{{.Title | htmlEscape}}</Title>
        <Source>{{.OwnerName | htmlEscape}} (BV: {{.Bvid | htmlEscape}})</Source>
        <Copyright>{{.OwnerName | htmlEscape}}</Copyright>
    </ProgramInformation>
{{range $i, $period := .Periods}}
    <Period id="{{$i}}" duration="{{$period.Duration | formatDuration}}">
        <AdaptationSet id="{{mul $i 2}}" mimeType="video/mp4" contentType="video" segmentAlignment="true" width="{{$period.Width}}" height="{{$period.Height}}" frameRate="30">
            <Representation id="{{mul $i 2}}" bandwidth="5000000" codecs="avc1.640028" width="{{$period.Width}}" height="{{$period.Height}}">
                <BaseURL>/v1/proxy?url={{$period.VideoURL | urlEscape}}</BaseURL>
                <SegmentBase indexRange="0-0">
                    <Initialization range="0-0"/>
                </SegmentBase>
            </Representation>
        </AdaptationSet>

        <AdaptationSet id="{{add (mul $i 2) 1}}" mimeType="audio/mp4" contentType="audio" segmentAlignment="true" lang="und">
            <Representation id="{{add (mul $i 2) 1}}" bandwidth="128000" codecs="mp4a.40.2" audioSamplingRate="44100">
                <AudioChannelConfiguration schemeIdUri="urn:mpeg:dash:23003:3:audio_channel_configuration:2011" value="2"/>
                <BaseURL>/v1/proxy?url={{$period.AudioURL | urlEscape}}</BaseURL>
                <SegmentBase indexRange="0-0">
                    <Initialization range="0-0"/>
                </SegmentBase>
            </Representation>
        </AdaptationSet>
    </Period>
{{end}}
</MPD>`

var mpdTemplate = template.Must(
	template.New("mpd").
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
		Parse(MPD_TEMPLATE),
)

type mpdData struct {
	Title         string
	OwnerName     string
	Bvid          string
	TotalDuration int
	Periods       []periodData
}

type periodData struct {
	Duration int
	Width    int
	Height   int
	VideoURL string
	AudioURL string
}

func apiPlay(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Empty id")
		return
	}

	p := r.URL.Query().Get("p")

	log.Info().
		Str("id", id).
		Str("p", p).
		Str("user-agent", r.Header.Get("User-Agent")).
		Msg("MPD request")

	pageNum := 1
	if p != "" {
		var err error
		pageNum, err = strconv.Atoi(p)
		if err != nil || pageNum < 1 {
			log.Error().
				Err(err).
				Int("pageNum", pageNum).
				Msg("Invalid page num")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Invalid page num: %d", pageNum)
			return
		}
	}

	var vInfo *biligo.VideoInfo
	vInfo, cached := getCachedVideoInfo(id)
	if !cached {
		var err error
		info, err := biligo.FetchVideoInfo(id)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Failed to fetch video info")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to fetch video info: %v", err)
			return
		}
		vInfo = &info
		setCachedVideoInfo(id, vInfo)
		log.Debug().Str("id", id).Msg("Video info cached")
	} else {
		log.Debug().Str("id", id).Msg("Video info from cache")
	}

	pages := vInfo.Pages

	if pageNum > len(pages) {
		log.Warn().
			Int("pageNum", pageNum).
			Int("len(pages)", len(pages)).
			Msg("Page num out of range")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Page num %d out of range %d", pageNum, len(pages))
		return
	}

	page := vInfo.Pages[pageNum-1]

	playurls, err := biligo.FetchVideoPlayurl(id, strconv.Itoa(page.Cid), biligo.VIDED_FNVAL_DASHALL)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to fetch video playurl")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Failed to fetch video playurl: %v", err)
		return
	}
	dash := playurls.Dash
	if dash == nil {
		log.Error().
			Msg("Failed to get dash info")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to get dash info")
		return
	}

	var selectedStream biligo.VideoPlayurlDashInfo

LOOP:
	for _, codecId := range codecPriority {
		bestQuality := -1
		for _, v := range dash.Video {
			if v.Codecid == codecId && v.Id <= maxQuality {
				if v.Id > bestQuality {
					selectedStream = v
					break LOOP
				}
			}
		}
	}

	if selectedStream.Id == 0 {
		selectedStream = dash.Video[0]
		log.Warn().
			Int("maxQuality", maxQuality).
			Int("available", selectedStream.Id).
			Msg("No video <= maxQuality found, using first available")
	}

	videoUrl := selectedStream.BackupUrl[0] // avoid pcdn
	// [TODO] audio quality selection
	audioUrl := dash.Audio[0].BackupUrl[0]
	log.Info().
		Int("codecid", selectedStream.Codecid).
		Int("quality", selectedStream.Id).
		Str("codecs", selectedStream.Codecs).
		Msg("Selected video stream")

	title := vInfo.Title
	if len(vInfo.Pages) > 1 && page.Part != "" {
		title = fmt.Sprintf("%s - %s", vInfo.Title, page.Part)
	}

	data := mpdData{
		Title:         title,
		OwnerName:     vInfo.Owner.Name,
		Bvid:          vInfo.Bvid,
		TotalDuration: page.Duration,
		Periods: []periodData{{
			Duration: page.Duration,
			Width:    page.Dimension.Width,
			Height:   page.Dimension.Height,
			VideoURL: videoUrl,
			AudioURL: audioUrl,
		}},
	}

	w.Header().Set("Content-Type", "application/dash+xml")
	w.Header().Set("Cache-Control", "public, max-age=300") // Cache for 5 minutes
	if err := mpdTemplate.Execute(w, data); err != nil {
		log.Error().Err(err).Msg("Failed to execute MPD template")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
