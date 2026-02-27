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
		host = server.Addr
		if host[0] == ':' {
			host = "localhost" + host
		} else if len(host) >= 7 && host[:7] == "0.0.0.0" {
			host = "localhost" + host[7:]
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	var items []M3u8Item
	maxDuration := 0
	for i, page := range pages {
		partTitle := page.Part
		if partTitle == "" {
			partTitle = fmt.Sprintf("P%d", i+1)
		}

		if page.Duration > maxDuration {
			maxDuration = page.Duration
		}

		items = append(items, M3u8Item{
			Duration: page.Duration,
			Title:    partTitle,
			URL:      fmt.Sprintf("%s://%s/v1/play/%s?p=%d", scheme, host, id, i+1),
		})
	}

	data := M3u8Data{
		Title:       vInfo.Title,
		MaxDuration: maxDuration,
		Items:       items,
	}

	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.m3u8\"", id))

	err := M3u8Template.Execute(w, data)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to execute M3U8 template")
		w.WriteHeader(http.StatusInternalServerError)
	}
}
