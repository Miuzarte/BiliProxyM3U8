package main

import (
	"fmt"
	"net/http"

	. "BProxy/templates"

	"github.com/Miuzarte/biligo"
	"github.com/rs/zerolog/log"
)

func apiVideo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Empty id")
		return
	}

	log.Info().
		Str("id", id).
		Str("user-agent", r.Header.Get("User-Agent")).
		Msg("M3U8 playlist request")

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
	}

	pages := vInfo.Pages

	host := r.Host
	if host == "" {
		host = "localhost:2233"
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	// Build M3U8 items
	var items []M3u8Item
	for i, page := range pages {
		partTitle := page.Part
		if partTitle == "" {
			partTitle = fmt.Sprintf("P%d", i+1)
		}

		items = append(items, M3u8Item{
			Duration: page.Duration,
			Title:    partTitle,
			URL:      fmt.Sprintf("%s://%s/v1/play/%s?p=%d", scheme, host, id, i+1),
		})
	}

	data := M3u8Data{
		Title: vInfo.Title,
		Items: items,
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.m3u8\"", id))

	if err := M3u8Template.Execute(w, data); err != nil {
		log.Error().Err(err).Msg("Failed to execute M3U8 template")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
