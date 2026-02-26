package main

import (
	"strings"

	"github.com/Miuzarte/biligo"
	"github.com/rs/zerolog/log"
)

func parseQuality(qualityStr string) int {
	qualityStr = strings.TrimSpace(strings.ToUpper(qualityStr))
	switch qualityStr {
	case "8K":
		return biligo.VIDEO_QN_8K
	case "DOLBY":
		return biligo.VIDEO_QN_DOLBY
	case "HDR":
		return biligo.VIDEO_QN_HDR
	case "4K":
		return biligo.VIDEO_QN_4K
	case "1080P60":
		return biligo.VIDEO_QN_1080P60
	case "1080P+", "1080PPLUS":
		return biligo.VIDEO_QN_1080PLUS
	case "1080P":
		return biligo.VIDEO_QN_1080
	case "720P60":
		return biligo.VIDEO_QN_720P60
	case "720P":
		return biligo.VIDEO_QN_720
	case "480P":
		return biligo.VIDEO_QN_480
	case "360P":
		return biligo.VIDEO_QN_360
	case "240P":
		return biligo.VIDEO_QN_240
	default:
		log.Warn().Str("quality", qualityStr).Msg("Unknown quality, using 8K")
		return biligo.VIDEO_QN_8K
	}
}

func parseCodecPriority(priorityStr string) []int {
	parts := strings.Split(priorityStr, ",")
	var codecs []int
	seen := make(map[int]bool)

	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		var codecID int

		switch {
		case strings.Contains(part, "av1") || strings.Contains(part, "av01"):
			codecID = biligo.VIDEO_CODEC_ID_AV1

		case strings.Contains(part, "hevc") || strings.Contains(part, "h265") || strings.Contains(part, "h.265"):
			codecID = biligo.VIDEO_CODEC_ID_HEVC

		case strings.Contains(part, "avc") || strings.Contains(part, "h264") || strings.Contains(part, "h.264"):
			codecID = biligo.VIDEO_CODEC_ID_AVC

		default:
			log.Warn().Str("codec", part).Msg("Unknown codec in priority")
			continue
		}

		if !seen[codecID] {
			codecs = append(codecs, codecID)
			seen[codecID] = true
		}
	}

	if len(codecs) == 0 {
		log.Warn().Msg("No valid codecs in priority, using default: \"HEVC, AVC, AV1\"")
		return []int{biligo.VIDEO_CODEC_ID_HEVC, biligo.VIDEO_CODEC_ID_AVC, biligo.VIDEO_CODEC_ID_AV1}
	}

	return codecs
}
