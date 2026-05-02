package main

import (
	"fmt"
	"strings"
)

func formatConfigMessage(cfg Config) string {
	var sb strings.Builder

	// استفاده از MarkdownV2 برای قالب‌بندی بهتر در تلگرام
	// نکته: کاراکترهای ویژه باید با \ فرار داده شوند
	escapedURL := escapeMarkdownV2(cfg.RawURL)

	// سرور و پورت (با فرض اینکه در cfg ذخیره شده باشد، در غیر این صورت می‌توان دوباره پارس کرد)
	server, port, _ := parseSlipnetURL(cfg.RawURL)

	// ایموجی‌ها: 🟢 برای فعال، 🟠 برای متوسط، 🔴 برای کند
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

// escapeMarkdownV2 کاراکترهای ویژه MarkdownV2 را فرار می‌دهد
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
