package main

import (
	"io"
	"maps"
	"net/http"
	"net/url"

	"github.com/Miuzarte/biligo"
	"github.com/rs/zerolog/log"
)

func apiProxy(w http.ResponseWriter, r *http.Request) {
	realURL, _ := url.QueryUnescape(r.URL.Query().Get("url"))

	rangeHeader := r.Header.Get("Range")
	log.Debug().
		Str("url", realURL).
		Str("range", rangeHeader).
		Str("user-agent", r.Header.Get("User-Agent")).
		Msg("Proxy request")

	req, _ := http.NewRequest("GET", realURL, nil)
	for k, v := range biligo.DefaultHeaders {
		req.Header.Set(k, v)
	}

	if rangeHeader != "" {
		req.Header.Set("Range", rangeHeader)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error().
			Err(err).
			Str("url", realURL).
			Msg("Proxy request failed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.Debug().
		Int("status", resp.StatusCode).
		Str("content-range", resp.Header.Get("Content-Range")).
		Str("content-length", resp.Header.Get("Content-Length")).
		Str("accept-ranges", resp.Header.Get("Accept-Ranges")).
		Msg("Proxy response")

	maps.Copy(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
