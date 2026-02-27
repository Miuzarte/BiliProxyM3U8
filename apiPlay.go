package main

import (
	"fmt"
	"net/http"
	"strconv"

	. "BProxy/templates"

	"github.com/Miuzarte/biligo"
	"github.com/rs/zerolog/log"
)

type mpdData struct {
	Title         string
	OwnerName     string
	Bvid          string
	TotalDuration int
	Periods       []periodData
}

type periodData struct {
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
	selectedAudio := dash.Audio[0]
	audioUrl := selectedAudio.BackupUrl[0]

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
			Duration:        page.Duration,
			VideoURL:        videoUrl,
			VideoMimeType:   selectedStream.MimeType,
			VideoCodecs:     selectedStream.Codecs,
			VideoBandwidth:  selectedStream.Bandwidth,
			VideoWidth:      selectedStream.Width,
			VideoHeight:     selectedStream.Height,
			VideoFrameRate:  selectedStream.FrameRate,
			VideoInitRange:  selectedStream.SegmentBase.Initialization,
			VideoIndexRange: selectedStream.SegmentBase.IndexRange,
			AudioURL:        audioUrl,
			AudioMimeType:   selectedAudio.MimeType,
			AudioCodecs:     selectedAudio.Codecs,
			AudioBandwidth:  selectedAudio.Bandwidth,
			AudioInitRange:  selectedAudio.SegmentBase.Initialization,
			AudioIndexRange: selectedAudio.SegmentBase.IndexRange,
		}},
	}

	w.Header().Set("Content-Type", "application/dash+xml")
	// Cache for 5 minutes
	// w.Header().Set("Cache-Control", "public, max-age=300")
	if err := MpdTemplate.Execute(w, data); err != nil {
		log.Error().Err(err).Msg("Failed to execute MPD template")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
