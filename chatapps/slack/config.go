package slack

type Config struct {
	BotToken      string
	AppToken      string
	SigningSecret string
	ServerAddr    string
	SystemPrompt  string
}
