package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Miuzarte/biligo"
	"github.com/mdp/qrterminal/v3"
	"github.com/rs/zerolog/log"
)

const IDENTITY_FILENAME = `bilibili_identity`

func loadIdentity() bool {
	f, err := os.OpenFile(IDENTITY_FILENAME, os.O_RDONLY, 0o600)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().
				Msg("bilibili_identity not exist, need login")
		} else {
			log.Error().
				Err(err).
				Msg("Failed to open identity file")
		}
		return false
	}
	defer f.Close()

	id := biligo.Identity{}
	err = json.NewDecoder(f).Decode(&id)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to decode identity")
		return false
	}
	biligo.ImportIdentity(id)
	log.Info().
		Msg("Identity loaded successfully")
	return true
}

func saveIdentity() error {
	id := biligo.ExportIdentity()
	f, err := os.OpenFile(IDENTITY_FILENAME, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		log.Error().Err(err).Msg("Failed to open identity file for writing")
		return err
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(id)
	if err != nil {
		log.Error().Err(err).Msg("Failed to encode identity")
		return err
	}
	return nil
}

func qrcodeLogin(ctx context.Context) error {
	qrcodeUrl, it, err := biligo.Login(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch QR code")
		return fmt.Errorf("failed to fetch qrcode: %w", err)
	}

	log.Info().
		Msg("Scan the QR code with Bilibili app:")
	fmt.Println()
	qrterminal.Generate(qrcodeUrl, qrterminal.L, os.Stdout)
	fmt.Println()

	for code, err := range it {
		if err != nil {
			if err == context.Canceled {
				return nil
			}
			log.Error().
				Err(err).
				Msg("Failed to poll login status")
			return fmt.Errorf("failed to poll login status: %w", err)
		}
		switch code {
		case biligo.LOGIN_CODE_STATE_SUCCESS:
			log.Info().
				Msg("Login successful!")
			err := saveIdentity()
			if err != nil {
				return fmt.Errorf("failed to save identity: %w", err)
			}
			return nil
		case biligo.LOGIN_CODE_STATE_EXPIRED:
			log.Warn().
				Msg("QR code expired")
			return fmt.Errorf("qr code expired")
		case biligo.LOGIN_CODE_STATE_SCANED:
			log.Info().
				Msg("QR code scanned, waiting for confirmation...")
		}
	}
	return fmt.Errorf("login flow ended unexpectedly")
}
