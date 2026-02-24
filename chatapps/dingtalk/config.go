package dingtalk

type Config struct {
	AppID         string
	AppSecret     string
	CallbackURL   string
	CallbackToken string
	CallbackKey   string
	ServerAddr    string
	MaxMessageLen int
	SystemPrompt  string
}
