package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// --- مدل داده‌ها ---
type Config struct {
	RawURL      string        `json:"raw_url"`
	Latency     time.Duration `json:"latency"`
	HealthCheck bool          `json:"health_check"`
	TLSStatus   bool          `json:"tls_status"`
	Steadiness  bool          `json:"steadiness"`
	Valid       bool          `json:"valid"`
}

// --- تابع اعتبارسنجی اصلی ---
func validateSlipnetConfig(rawURL string) Config {
	// این تابع باید به Checker ارجاع دهد
	return CheckerValidateSlipnetConfig(rawURL)
}

// --- تابع کمکی برای تجزیه رشته‌های کانال ---
func parseChannels(channelsStr string) []string {
	if channelsStr == "" {
		return []string{}
	}
	channelsStr = strings.Trim(channelsStr, "[]")
	parts := strings.Split(channelsStr, ",")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
		parts[i] = strings.Trim(parts[i], "\"")
	}
	return parts
}

func main() {
	// 1. خواندن متغیرهای محیطی
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	channelsEnv := os.Getenv("TELEGRAM_CHANNELS")    // فرمت: ["@channel1","@channel2"]
	chatIDStr := os.Getenv("TELEGRAM_CHANNEL_ID")    // شناسه عددی کانال مقصد

	if botToken == "" || channelsEnv == "" || chatIDStr == "" {
		log.Fatal("Missing required environment variables: TELEGRAM_BOT_TOKEN, TELEGRAM_CHANNELS, TELEGRAM_CHANNEL_ID")
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid TELEGRAM_CHANNEL_ID: %v", err)
	}

	channels := parseChannels(channelsEnv)
	if len(channels) == 0 {
		log.Fatal("No channels to scrape")
	}
	log.Printf("Channels to scrape: %v", channels)
	log.Printf("Target channel ID: %d", chatID)

	// 2. راه‌اندازی ربات تلگرام
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// 3. اسکرپ کانال‌ها و استخراج کانفیگ‌ها
	log.Println("Starting to scrape channels...")
	allConfigs := []Config{}

	for _, channel := range channels {
		log.Printf("Scraping channel: %s", channel)
		content := ScrapeChannel(channel)
		if content == "" {
			log.Printf("No content scraped from %s", channel)
			continue
		}
		urls := ExtractSlipnetURLs(content)
		log.Printf("Found %d slipnet config(s) in %s", len(urls), channel)

		for _, rawURL := range urls {
			cfg := validateSlipnetConfig(rawURL)
			allConfigs = append(allConfigs, cfg)
		}
	}
	log.Printf("Total configs collected: %d", len(allConfigs))

	if len(allConfigs) == 0 {
		log.Println("No valid slipnet configs found. Exiting.")
		return
	}

	// 4. فیلتر کردن کانفیگ‌های معتبر
	validConfigs := []Config{}
	for _, cfg := range allConfigs {
		if cfg.Valid {
			validConfigs = append(validConfigs, cfg)
		}
	}
	log.Printf("Valid configs after health check: %d", len(validConfigs))

	if len(validConfigs) == 0 {
		log.Println("No valid configs passed health check. Exiting.")
		return
	}

	// 5. ارسال کانفیگ‌های معتبر به کانال تلگرام
	for _, cfg := range validConfigs {
		msgText := formatConfigMessage(cfg)
		msg := tgbotapi.NewMessage(chatID, msgText)
		msg.ParseMode = tgbotapi.ModeMarkdownV2
		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Failed to send message: %v", err)
		} else {
			log.Printf("Sent config: %s", cfg.RawURL)
		}
		time.Sleep(2 * time.Second) // جلوگیری از ریت لیمیت
	}

	// 6. ذخیره گزارش نهایی
	report := struct {
		Timestamp   time.Time `json:"timestamp"`
		TotalFound  int       `json:"total_found"`
		ValidCount  int       `json:"valid_count"`
		ChannelList []string  `json:"channel_list"`
	}{
		Timestamp:   time.Now(),
		TotalFound:  len(allConfigs),
		ValidCount:  len(validConfigs),
		ChannelList: channels,
	}
	reportJSON, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile("report.json", reportJSON, 0644); err != nil {
		log.Printf("Failed to write report: %v", err)
	} else {
		log.Println("Report saved to report.json")
	}

	// 7. منتظر سیگنال خاتمه برای بسته شدن صحیح
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down gracefully...")
}
