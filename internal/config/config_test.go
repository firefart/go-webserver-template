package config

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	public, err := os.CreateTemp("", "test")
	require.Nil(t, err)
	defer func(public *os.File) {
		err := public.Close()
		if err != nil {
			require.Nil(t, err)
		}
	}(public)
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			require.Nil(t, err)
		}
	}(public.Name())
	private, err := os.CreateTemp("", "test")
	require.Nil(t, err)
	defer func(private *os.File) {
		err := private.Close()
		if err != nil {
			require.Nil(t, err)
		}
	}(private)
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			require.Nil(t, err)
		}
	}(private.Name())

	config := fmt.Sprintf(`{
  "server": {
    "listen": "127.0.0.1:8000",
    "listen_pprof": "127.0.0.1:1234",
    "graceful_timeout": "5s",
    "cloudflare": false,
    "secret_key_header_name": "X-Secret-Key-Header",
    "secret_key_header_value": "SECRET",
    "tls": {
      "public_key": "%[1]s",
      "private_key": "%[2]s",
      "mtls_root_ca": "%[1]s",
      "mtls_cert_subject": "Subject"
    }
  },
  "cache": {
    "enabled": true,
    "timeout": "1h"
  },
  "timeout": "5s",
  "mail": {
    "enabled": true,
    "server": "server.com",
    "port": 25,
    "from": {
      "name": "From",
      "email": "user@domain.com"
    },
    "to": [
      "user1@domain.com",
      "user2@domain.com"
    ],
    "user": "username",
    "password": "password",
    "tls": false,
    "starttls": true,
    "skiptls": false,
    "retries": 5,
    "timeout": "5s"
  },
  "database": {
    "filename": "data.db"
  },
  "notifications": {
    "telegram": {
      "enabled": true,
      "api_token": "token",
      "chat_ids": [
        1,
        2
      ]
    },
    "discord": {
      "enabled": true,
      "bot_token": "token",
      "oauth_token": "",
      "channel_ids": [
        "1",
        "2"
      ]
    },
    "email": {
      "enabled": true,
      "sender": "test@test.com",
      "server": "smtp.server.com",
      "port": 25,
      "username": "user",
      "password": "pass",
      "recipients": [
        "test@test.com",
        "a@a.com"
      ]
    },
    "sendgrid": {
      "enabled": true,
      "api_key": "apikey",
      "sender_address": "test@test.com",
      "sender_name": "Test",
      "recipients": [
        "test@test.com",
        "a@a.com"
      ]
    },
    "msteams": {
      "enabled": true,
      "webhooks": [
        "https://url1.com",
        "https://url2.com"
      ]
    }
  }
}`, public.Name(), private.Name())

	f, err := os.CreateTemp("", "config")
	require.Nil(t, err)
	tmpFilename := f.Name()
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			require.Nil(t, err)
		}
	}(f)
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			require.Nil(t, err)
		}
	}(tmpFilename)
	_, err = f.WriteString(config)
	require.Nil(t, err)

	c, err := GetConfig(tmpFilename)
	require.Nil(t, err)

	require.Equal(t, "127.0.0.1:8000", c.Server.Listen)
	require.Equal(t, "127.0.0.1:1234", c.Server.PprofListen)
	require.Equal(t, 5*time.Second, c.Server.GracefulTimeout)
	require.Equal(t, false, c.Server.Cloudflare)
	require.Equal(t, "X-Secret-Key-Header", c.Server.SecretKeyHeaderName)
	require.Equal(t, "SECRET", c.Server.SecretKeyHeaderValue)
	require.Equal(t, public.Name(), c.Server.TLS.MTLSRootCA)
	require.Equal(t, "Subject", c.Server.TLS.MTLSCertSubject)
	require.Equal(t, public.Name(), c.Server.TLS.PublicKey)
	require.Equal(t, private.Name(), c.Server.TLS.PrivateKey)

	require.Equal(t, 5*time.Second, c.Timeout)

	require.Equal(t, true, c.Cache.Enabled)
	require.Equal(t, 1*time.Hour, c.Cache.Timeout)

	require.Equal(t, true, c.Mail.Enabled)
	require.Equal(t, "server.com", c.Mail.Server)
	require.Equal(t, 25, c.Mail.Port)
	require.Equal(t, "From", c.Mail.From.Name)
	require.Equal(t, "user@domain.com", c.Mail.From.Mail)
	require.Len(t, c.Mail.To, 2)
	require.Equal(t, "user1@domain.com", c.Mail.To[0])
	require.Equal(t, "user2@domain.com", c.Mail.To[1])
	require.Equal(t, "username", c.Mail.User)
	require.Equal(t, "password", c.Mail.Password)
	require.Equal(t, false, c.Mail.TLS)
	require.Equal(t, true, c.Mail.StartTLS)
	require.Equal(t, false, c.Mail.SkipTLS)
	require.Equal(t, 5, c.Mail.Retries)
	require.Equal(t, 5*time.Second, c.Mail.Timeout)

	require.Equal(t, "data.db", c.Database.Filename)

	require.Len(t, c.Notifications.Telegram.ChatIDs, 2)
	require.Equal(t, int64(1), c.Notifications.Telegram.ChatIDs[0])
	require.Equal(t, int64(2), c.Notifications.Telegram.ChatIDs[1])
	require.Equal(t, "token", c.Notifications.Telegram.APIToken)

	require.Len(t, c.Notifications.Discord.ChannelIDs, 2)
	require.Equal(t, "1", c.Notifications.Discord.ChannelIDs[0])
	require.Equal(t, "2", c.Notifications.Discord.ChannelIDs[1])
	require.Equal(t, "token", c.Notifications.Discord.BotToken)
	require.Equal(t, "", c.Notifications.Discord.OAuthToken)

	require.Equal(t, "test@test.com", c.Notifications.Email.Sender)
	require.Equal(t, "smtp.server.com", c.Notifications.Email.Server)
	require.Equal(t, 25, c.Notifications.Email.Port)
	require.Equal(t, "user", c.Notifications.Email.Username)
	require.Equal(t, "pass", c.Notifications.Email.Password)
	require.Len(t, c.Notifications.Email.Recipients, 2)
	require.Equal(t, "test@test.com", c.Notifications.Email.Recipients[0])
	require.Equal(t, "a@a.com", c.Notifications.Email.Recipients[1])

	require.Equal(t, "apikey", c.Notifications.SendGrid.APIKey)
	require.Equal(t, "test@test.com", c.Notifications.SendGrid.SenderAddress)
	require.Equal(t, "Test", c.Notifications.SendGrid.SenderName)
	require.Len(t, c.Notifications.SendGrid.Recipients, 2)
	require.Equal(t, "test@test.com", c.Notifications.SendGrid.Recipients[0])
	require.Equal(t, "a@a.com", c.Notifications.SendGrid.Recipients[1])

	require.Len(t, c.Notifications.MSTeams.Webhooks, 2)
	require.Equal(t, "https://url1.com", c.Notifications.MSTeams.Webhooks[0])
	require.Equal(t, "https://url2.com", c.Notifications.MSTeams.Webhooks[1])
}
