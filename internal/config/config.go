package config

import (
	"fmt"
	"time"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type Configuration struct {
	Server        Server        `koanf:"server"`
	Cache         Cache         `koanf:"cache"`
	Mail          Mail          `koanf:"mail"`
	Database      Database      `koanf:"database"`
	Notifications Notification  `koanf:"notifications"`
	Timeout       time.Duration `koanf:"timeout"`
}

type Server struct {
	Listen               string        `koanf:"listen"`
	PprofListen          string        `koanf:"listen_pprof"`
	GracefulTimeout      time.Duration `koanf:"graceful_timeout"`
	TLS                  TLS           `koanf:"tls"`
	Cloudflare           bool          `koanf:"cloudflare"`
	SecretKeyHeaderName  string        `koanf:"secret_key_header_name"`
	SecretKeyHeaderValue string        `koanf:"secret_key_header_value"`
}

type TLS struct {
	PublicKey       string `koanf:"public_key"`
	PrivateKey      string `koanf:"private_key"`
	MTLSRootCA      string `koanf:"mtls_root_ca"`
	MTLSCertSubject string `koanf:"mtls_cert_subject"`
}

type Cache struct {
	Enabled bool          `koanf:"enabled"`
	Timeout time.Duration `koanf:"timeout"`
}

type Mail struct {
	Enabled bool   `koanf:"enabled"`
	Server  string `koanf:"server"`
	Port    int    `koanf:"port"`
	From    struct {
		Name string `koanf:"name"`
		Mail string `koanf:"mail"`
	} `koanf:"from"`
	To       []string      `koanf:"to"`
	User     string        `koanf:"user"`
	Password string        `koanf:"password"`
	TLS      bool          `koanf:"tls"`
	StartTLS bool          `koanf:"starttls"`
	SkipTLS  bool          `koanf:"skiptls"`
	Retries  int           `koanf:"retries"`
	Timeout  time.Duration `koanf:"timeout"`
}

type Database struct {
	Filename string `koanf:"filename"`
}

type Notification struct {
	Telegram NotificationTelegram `koanf:"telegram"`
	Discord  NotificationDiscord  `koanf:"discord"`
	Email    NotificationEmail    `koanf:"email"`
	SendGrid NotificationSendGrid `koanf:"sendgrid"`
	MSTeams  NotificationMSTeams  `koanf:"msteams"`
}

type NotificationTelegram struct {
	APIToken string  `koanf:"api_token"`
	ChatIDs  []int64 `koanf:"chat_ids"`
}
type NotificationDiscord struct {
	BotToken   string   `koanf:"bot_token"`
	OAuthToken string   `koanf:"oauth_token"`
	ChannelIDs []string `koanf:"channel_ids"`
}

type NotificationEmail struct {
	Sender     string   `koanf:"sender"`
	Server     string   `koanf:"server"`
	Port       int      `koanf:"port"`
	Username   string   `koanf:"username"`
	Password   string   `koanf:"password"`
	Recipients []string `koanf:"recipients"`
}

type NotificationSendGrid struct {
	APIKey        string   `koanf:"api_key"`
	SenderAddress string   `koanf:"sender_address"`
	SenderName    string   `koanf:"sender_name"`
	Recipients    []string `koanf:"recipients"`
}

type NotificationMSTeams struct {
	Webhooks []string `koanf:"webhooks"`
}

var defaultConfig = Configuration{
	Server: Server{
		Listen:              "127.0.0.1:8000",
		PprofListen:         "127.0.0.1:1234",
		GracefulTimeout:     10 * time.Second,
		Cloudflare:          false,
		SecretKeyHeaderName: "X-Secret-Key-Header",
	},
	Cache: Cache{
		Enabled: true,
		Timeout: 1 * time.Hour,
	},
	Database: Database{
		Filename: "db.sqlite3",
	},
	Timeout: 5 * time.Second,
}

func GetConfig(f string) (Configuration, error) {
	k := koanf.NewWithConf(koanf.Conf{
		Delim: ".",
	})

	if err := k.Load(structs.Provider(defaultConfig, "koanf"), nil); err != nil {
		return Configuration{}, err
	}

	if err := k.Load(file.Provider(f), json.Parser()); err != nil {
		return Configuration{}, err
	}

	var config Configuration
	if err := k.Unmarshal("", &config); err != nil {
		return Configuration{}, err
	}

	if config.Server.SecretKeyHeaderName == "" {
		return Configuration{}, fmt.Errorf("please supply a secret key header name in the config")
	}

	if config.Server.SecretKeyHeaderValue == "" {
		return Configuration{}, fmt.Errorf("please supply a secret key header value in the config")
	}

	return config, nil
}
