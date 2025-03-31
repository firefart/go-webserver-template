package config

import (
	"errors"
	"fmt"
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
	Proxy         *Proxy        `koanf:"proxy"`
	Cache         Cache         `koanf:"cache"`
	Mail          Mail          `koanf:"mail"`
	Database      Database      `koanf:"database"`
	Notifications Notification  `koanf:"notifications"`
	Timeout       time.Duration `koanf:"timeout" validate:"required"`
	UserAgent     string        `koanf:"user_agent"`
}

type Server struct {
	Listen               string        `koanf:"listen" validate:"required,hostname_port"`
	PprofListen          string        `koanf:"listen_pprof" validate:"required,hostname_port"`
	MetricsListen        string        `koanf:"listen_metrics" validate:"required,hostname_port"`
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

type Proxy struct {
	URL      string `koanf:"url" json:"url" validate:"omitempty,url"`
	Username string `koanf:"username" json:"username" validate:"required_with=Password"`
	Password string `koanf:"password" json:"password" validate:"required_with=Username"`
	NoProxy  string `koanf:"no_proxy" json:"no_proxy"`
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
	APIToken string  `koanf:"api_token" validate:"required_if=Enabled true"`
	ChatIDs  []int64 `koanf:"chat_ids" validate:"required_if=Enabled true,dive"`
}
type NotificationDiscord struct {
	Enabled    bool     `koanf:"enabled"`
	BotToken   string   `koanf:"bot_token" validate:"required_without=OAuthToken,excluded_with=OAuthToken"`
	OAuthToken string   `koanf:"oauth_token" validate:"required_without=BotToken,excluded_with=BotToken"`
	ChannelIDs []string `koanf:"channel_ids" validate:"required_if=Enabled true,dive"`
}

type NotificationEmail struct {
	Enabled    bool     `koanf:"enabled"`
	Sender     string   `koanf:"sender" validate:"required_if=Enabled true,email"`
	Server     string   `koanf:"server" validate:"required_if=Enabled true,fqdn"`
	Port       int      `koanf:"port" validate:"required_if=Enabled true,gt=0,lte=65535"`
	Username   string   `koanf:"username"`
	Password   string   `koanf:"password"`
	Recipients []string `koanf:"recipients" validate:"required_if=Enabled true,dive,email"`
}

type NotificationSendGrid struct {
	Enabled       bool     `koanf:"enabled"`
	APIKey        string   `koanf:"api_key" validate:"required_if=Enabled true"`
	SenderAddress string   `koanf:"sender_address" validate:"required_if=Enabled true,email"`
	SenderName    string   `koanf:"sender_name" validate:"required_if=Enabled true"`
	Recipients    []string `koanf:"recipients" validate:"required_if=Enabled true,dive,email"`
}

type NotificationMSTeams struct {
	Enabled  bool     `koanf:"enabled"`
	Webhooks []string `koanf:"webhooks" validate:"required_if=Enabled true,dive,http_url"`
}

var defaultConfig = Configuration{
	Server: Server{
		Listen:              "127.0.0.1:8000",
		PprofListen:         "127.0.0.1:1234",
		MetricsListen:       "127.0.0.1:1235",
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

		var valErr validator.ValidationErrors
		if ok := errors.As(err, &valErr); !ok {
			return Configuration{}, fmt.Errorf("could not cast err to ValidationErrors: %w", err)
		}
		var resultErr error
		for _, err := range valErr {
			resultErr = multierror.Append(resultErr, err)
		}
		return Configuration{}, resultErr
	}

	return config, nil
}
