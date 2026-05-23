package main

import (
	"fmt"
	"github.com/joakim/fintrack-api/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("config error: %v\n", err)
		return
	}
	fmt.Printf("SMTP_HOST=%q\n", cfg.SMTPHost)
	fmt.Printf("SMTP_PORT=%d\n", cfg.SMTPPort)
	fmt.Printf("SMTP_USERNAME=%q\n", cfg.SMTPUsername)
	fmt.Printf("SMTP_FROM=%q\n", cfg.SMTPFrom)
	if cfg.SMTPPassword != "" {
		fmt.Printf("SMTP_PASSWORD=<set, len=%d>\n", len(cfg.SMTPPassword))
	} else {
		fmt.Printf("SMTP_PASSWORD=<empty>\n")
	}
}
