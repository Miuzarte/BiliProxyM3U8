package main

import (
	"fmt"
	"net/http"

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

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.m3u8\"", id))

	fmt.Fprintf(w, "#EXTM3U\n")
	fmt.Fprintf(w, "#PLAYLIST:%s\n", vInfo.Title)

	for i, page := range pages {
		partTitle := page.Part
		if partTitle == "" {
			partTitle = fmt.Sprintf("P%d", i+1)
		}

		// Use part title directly, not "Video Title - Part Title"
		fmt.Fprintf(w, "#EXTINF:%d,%s\n", page.Duration, partTitle)
		fmt.Fprintf(w, "%s://%s/v1/play/%s?p=%d\n", scheme, host, id, i+1)
	}
}
