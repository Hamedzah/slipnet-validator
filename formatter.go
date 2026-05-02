package main

import (
	"fmt"
	"strings"
	"time"
)

func formatConfigMessage(cfg Config) string {
	var sb strings.Builder

	escapedURL := escapeMarkdownV2(cfg.RawURL)

	server, port, _ := parseSlipnetURL(cfg.RawURL)

	latencyIcon := "🟢"
	if cfg.Latency > 1000*time.Millisecond {
		latencyIcon = "🟠"
	}
	if cfg.Latency > 2000*time.Millisecond {
		latencyIcon = "🔴"
	}

	sb.WriteString(fmt.Sprintf("`%s`\n\n", escapedURL))
	sb.WriteString(fmt.Sprintf("🌍 *سرور:* `%s`\n", server))
	sb.WriteString(fmt.Sprintf("🔌 *پورت:* `%d`\n", port))
	sb.WriteString(fmt.Sprintf("%s *تأخیر:* `%dms`\n", latencyIcon, cfg.Latency.Milliseconds()))
	sb.WriteString("🛡️ *TLS:* فعال\n")
	sb.WriteString(fmt.Sprintf("✅ *وضعیت:* %s\n", map[bool]string{true: "سالم", false: "نامعتبر"}[cfg.Valid]))

	sb.WriteString("\n✨ @SlipGate_hub\n")
	return sb.String()
}

func escapeMarkdownV2(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}
