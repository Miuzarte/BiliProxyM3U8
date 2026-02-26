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

var loginOnly = flag.Bool("login", false, "Only perform login and exit")

var (
	cwg    = contextWaitGroup.New(context.Background())
	server = http.Server{Addr: ":2233"}
)

func init() {
	flag.Parse()

	// If --login flag is set, only perform login and exit
	if *loginOnly {
		log.Info().
			Msg("Login mode: performing login only")
		if err := qrcodeLogin(context.Background()); err != nil {
			log.Fatal().
				Err(err).
				Msg("Login failed")
		}
		log.Info().
			Msg("Login completed successfully, exiting")
		return
	}
}

func main() {
	stop := cwg.WithSignal(syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	defer cwg.Cancel()
	defer stop()

	http.HandleFunc("GET /v1/video/{id}", apiVideo)
	http.HandleFunc("GET /v1/play/{id}", apiPlay)
	http.HandleFunc("GET /v1/proxy", apiProxy)

	loggedIn := loadIdentity()
	if !loggedIn {
		log.Warn().Msg("Not logged in. Starting QR code login in background...")
		cwg.Go(func(ctx context.Context) {
			if err := qrcodeLogin(ctx); err != nil {
				log.Error().
					Err(err).
					Msg("Background login failed")
			}
		})
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
