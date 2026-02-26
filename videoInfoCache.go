package main

import (
	"context"
	"sync"
	"time"

	"github.com/Miuzarte/biligo"
	"github.com/rs/zerolog/log"
)

type cacheEntry struct {
	data      any
	expiresAt time.Time
}

var (
	videoInfoCache = make(map[string]*cacheEntry)
	cacheMutex     sync.RWMutex
)

func getCachedVideoInfo(id string) (*biligo.VideoInfo, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	if entry, ok := videoInfoCache[id]; ok {
		if time.Now().Before(entry.expiresAt) {
			return entry.data.(*biligo.VideoInfo), true
		}
	}
	return nil, false
}

func setCachedVideoInfo(id string, info *biligo.VideoInfo) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	videoInfoCache[id] = &cacheEntry{
		data:      info,
		expiresAt: time.Now().Add(5 * time.Minute), // Cache for 5 minutes
	}
}

func cleanupExpiredCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	now := time.Now()
	for id, entry := range videoInfoCache {
		if now.After(entry.expiresAt) {
			delete(videoInfoCache, id)
		}
	}
}

func startCacheCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cleanupExpiredCache()
			log.Debug().Msg("Cache cleanup completed")
		case <-ctx.Done():
			return
		}
	}
}
