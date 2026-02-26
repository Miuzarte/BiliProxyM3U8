package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"BProxy/contextWaitGroup"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:             os.Stdout,
		FormatTimestamp: logFormatTimestamp,
		FormatLevel:     logFormatLevel,
	})
	// zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

var (
	fLoginOnly = flag.Bool("login", false,
		"Only perform login and exit")
	fCodecPriority = flag.String("codec", "hevc,avc,av1",
		"Codec priority (av1/av01, hevc/h265/h.265, avc/h264/h.264)")
	fQuality = flag.String("quality", "1080P",
		"Maximum quality (8K, DOLBY, HDR, 4K, 1080P60, 1080P+, 1080P, 720P60, 720P, 480P, 360P, 240P)")
)

var (
	cwg    = contextWaitGroup.New(context.Background())
	server = http.Server{Addr: ":2233"}
)

var (
	maxQuality    int
	codecPriority []int
)

func init() {
	flag.Parse()

	maxQuality = parseQuality(*fQuality)
	codecPriority = parseCodecPriority(*fCodecPriority)

	log.Info().
		Int("maxQuality", maxQuality).
		Ints("codecPriority", codecPriority).
		Msg("Video selection preferences loaded")
}

func main() {
	stop := cwg.WithSignal(syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer cwg.Cancel()
	defer stop()

	http.HandleFunc("GET /v1/video/{id}", apiVideo)
	http.HandleFunc("GET /v1/play/{id}", apiPlay)
	http.HandleFunc("GET /v1/proxy", apiProxy)

	switch {
	case !loadIdentity():
		log.Warn().
			Msg("Not logged in. Starting QR code login in background...")
		fallthrough
	case *fLoginOnly:
		cwg.Go(func(ctx context.Context) {
			if err := qrcodeLogin(ctx); err != nil {
				log.Error().
					Err(err).
					Msg("Background login failed")
			}
		})
	}
	if *fLoginOnly {
		// only perform login and exit
		cwg.Wait()
		return
	}

	cwg.Go(func(ctx context.Context) {
		startCacheCleanup(ctx)
	})

	cwg.Go(func(_ context.Context) {
		log.Info().
			Str("addr", server.Addr).
			Msg("Service running")
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Fatal().
				Err(err).
				Msg("Http server error")
		}
	})

	cwg.Go(func(ctx context.Context) {
		<-ctx.Done()
		stop() // 停止捕获信号

		log.Info().
			Msg("Shutting down server..., Ctrl+C again to force quit")
		shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
		defer shutdownCancel()
		shutdownCtx, stop := signal.NotifyContext(shutdownCtx, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
		defer stop()

		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Error().
				Err(err).
				Msg("Server shutdown error")
		} else {
			log.Info().
				Msg("Server stopped gracefully")
		}
	})

	cwg.Wait()
}
