package totputil

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
)

const (
	backupCodeCount  = 8
	backupCodeLength = 5
	issuer           = "FinTrack"
)

// GenerateSecret returns a base32-encoded TOTP secret.
func GenerateSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// QRCodeURI returns the otpauth:// URI suitable for QR code rendering.
func QRCodeURI(secret, accountName, issuerName string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuerName, accountName, secret, issuerName)
}

// Validate checks the given code against the TOTP secret with a 1-window drift tolerance.
func Validate(secret, code string) bool {
	ok, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period: 30,
		Skew:   1,
		Digits: 6,
	})
	return ok && err == nil
}

// GenerateBackupCodes returns 8 one-time recovery codes in "XXXXX-XXXXX" format.
func GenerateBackupCodes() ([]string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codes := make([]string, backupCodeCount)
	for i := range codes {
		var parts [2]string
		for p := range parts {
			buf := make([]byte, backupCodeLength)
			for j := range buf {
				n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
				if err != nil {
					return nil, err
				}
				buf[j] = chars[n.Int64()]
			}
			parts[p] = string(buf)
		}
		codes[i] = strings.Join(parts[:], "-")
	}
	return codes, nil
}
