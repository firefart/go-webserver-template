package config

import (
	"errors"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"

	"github.com/go-playground/validator/v10"
)

type Configuration struct {
	Server        Server        `koanf:"server"`
	Cache         Cache         `koanf:"cache"`
	Mail          Mail          `koanf:"mail"`
	Database      Database      `koanf:"database"`
	Notifications Notification  `koanf:"notifications"`
	Timeout       time.Duration `koanf:"timeout" validate:"required"`
}

type Server struct {
	Listen               string        `koanf:"listen" validate:"required,hostname_port"`
	PprofListen          string        `koanf:"listen_pprof" validate:"required,hostname_port"`
	GracefulTimeout      time.Duration `koanf:"graceful_timeout" validate:"required"`
	Cloudflare           bool          `koanf:"cloudflare"`
	SecretKeyHeaderName  string        `koanf:"secret_key_header_name" validate:"required"`
	SecretKeyHeaderValue string        `koanf:"secret_key_header_value" validate:"required"`
	TLS                  TLS           `koanf:"tls"`
}

type TLS struct {
	PublicKey       string `koanf:"public_key" validate:"omitempty,file"`
	PrivateKey      string `koanf:"private_key" validate:"omitempty,file"`
	MTLSRootCA      string `koanf:"mtls_root_ca" validate:"omitempty,file"`
	MTLSCertSubject string `koanf:"mtls_cert_subject"`
}

type Cache struct {
	Enabled bool          `koanf:"enabled"`
	Timeout time.Duration `koanf:"timeout" validate:"required"`
}

type Mail struct {
	Enabled bool   `koanf:"enabled"`
	Server  string `koanf:"server" validate:"required"`
	Port    int    `koanf:"port" validate:"required,gt=0,lte=65535"`
	From    struct {
		Name string `koanf:"name" validate:"required"`
		Mail string `koanf:"email" validate:"required,email"`
	} `koanf:"from"`
	To       []string      `koanf:"to" validate:"required,dive,email"`
	User     string        `koanf:"user"`
	Password string        `koanf:"password"`
	TLS      bool          `koanf:"tls"`
	StartTLS bool          `koanf:"starttls"`
	SkipTLS  bool          `koanf:"skiptls"`
	Retries  int           `koanf:"retries" validate:"required"`
	Timeout  time.Duration `koanf:"timeout" validate:"required"`
}

type Database struct {
	Filename string `koanf:"filename" validate:"required"`
}

type Notification struct {
	Telegram NotificationTelegram `koanf:"telegram"`
	Discord  NotificationDiscord  `koanf:"discord"`
	Email    NotificationEmail    `koanf:"email"`
	SendGrid NotificationSendGrid `koanf:"sendgrid"`
	MSTeams  NotificationMSTeams  `koanf:"msteams"`
}

type NotificationTelegram struct {
	Enabled  bool    `koanf:"enabled"`
	APIToken string  `koanf:"api_token"`
	ChatIDs  []int64 `koanf:"chat_ids"`
}
type NotificationDiscord struct {
	Enabled    bool     `koanf:"enabled"`
	BotToken   string   `koanf:"bot_token"`
	OAuthToken string   `koanf:"oauth_token"`
	ChannelIDs []string `koanf:"channel_ids"`
}

type NotificationEmail struct {
	Enabled    bool     `koanf:"enabled"`
	Sender     string   `koanf:"sender"`
	Server     string   `koanf:"server"`
	Port       int      `koanf:"port"`
	Username   string   `koanf:"username"`
	Password   string   `koanf:"password"`
	Recipients []string `koanf:"recipients"`
}

type NotificationSendGrid struct {
	Enabled       bool     `koanf:"enabled"`
	APIKey        string   `koanf:"api_key"`
	SenderAddress string   `koanf:"sender_address"`
	SenderName    string   `koanf:"sender_name"`
	Recipients    []string `koanf:"recipients"`
}

type NotificationMSTeams struct {
	Enabled  bool     `koanf:"enabled"`
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
	validate := validator.New(validator.WithRequiredStructEnabled())

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

	if err := validate.Struct(config); err != nil {
		var invalidValidationError *validator.InvalidValidationError
		if errors.As(err, &invalidValidationError) {
			return Configuration{}, err
		}

		var resultErr error
		for _, err := range err.(validator.ValidationErrors) {
			// TODO: create new error with own message
			resultErr = multierror.Append(resultErr, err)
		}
		return Configuration{}, resultErr
	}

	return config, nil
}
