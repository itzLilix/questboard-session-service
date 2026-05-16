package validation

import (
	"errors"
	"net/url"
	"strings"
)

var (
	ErrInvalidURL = errors.New("invalid url")
	ErrHostNotAllowed = errors.New("host not in allowlist")
)

var allowedAvatarHosts = map[string]bool{
	"i.imgur.com":           true,
	"imgur.com":             true,
	"cdn.discordapp.com":    true,
	"media.discordapp.net":  true,
	"gyazo.com":             true,
	"i.gyazo.com":           true,
}

var allowedSheetHosts = map[string]bool{
	"www.dndbeyond.com":  true,
	"dndbeyond.com":      true,
	"foundryvtt.com":     true,
	"forge-vtt.com":      true,
	"app.roll20.net":     true,
	"roll20.net":         true,
	"docs.google.com":    true,
	"drive.google.com":   true,
	"notion.so":          true,
	"www.notion.so":      true,
	"dropbox.com":        true,
	"www.dropbox.com":    true,
	"1drv.ms":            true,
	"onedrive.live.com":  true,
	"pathbuilder2e.com":  true,
	"dicecloud.com":      true,
	"dicecloud.app":      true,
	"imgur.com":          true,
	"i.imgur.com":        true,
	"worldanvil.com":     true,
}

func ValidateAvatarURL(raw string) error {
	return validateAgainst(raw, allowedAvatarHosts)
}

func ValidateSheetURL(raw string) error {
	return validateAgainst(raw, allowedSheetHosts)
}

func validateAgainst(raw string, allowed map[string]bool) error {
	if raw == "" {
		return nil
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ErrInvalidURL
	}
	if u.Scheme != "https" {
		return ErrInvalidURL
	}
	host := strings.ToLower(u.Hostname())
	if !allowed[host] {
		return ErrHostNotAllowed
	}
	return nil
}
