
package config

import (
    "os"
    "strings"
)

type Config struct {
    TelegramToken string
    DataDir       string
}

func Load() Config {
    token := readSecret("/run/secrets/telegram_bot_token")
    if token == "" {
        token = strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
    }

    return Config{
        TelegramToken: token,
        DataDir:       "./data",
    }
}

func readSecret(path string) string {
    b, err := os.ReadFile(path)
    if err != nil {
        return ""
    }
    return strings.TrimSpace(string(b))
}
