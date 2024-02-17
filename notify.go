package main

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"

	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/discord"
	"github.com/nikoksr/notify/service/mail"
	"github.com/nikoksr/notify/service/msteams"
	"github.com/nikoksr/notify/service/sendgrid"
	"github.com/nikoksr/notify/service/telegram"
)

func setupNotifications(config Configuration, logger *slog.Logger) (*notify.Notify, error) {
	not := notify.New()
	var services []notify.Notifier

	if config.Notifications.Telegram.APIToken != "" {
		logger.Info("Notifications: using telegram")
		telegramService, err := telegram.New(config.Notifications.Telegram.APIToken)
		if err != nil {
			return nil, fmt.Errorf("telegram setup: %w", err)
		}
		telegramService.AddReceivers(config.Notifications.Telegram.ChatIDs...)
		services = append(services, telegramService)
	}

	if config.Notifications.Discord.BotToken != "" || config.Notifications.Discord.OAuthToken != "" {
		logger.Info("Notifications: using discord")
		discordService := discord.New()
		if config.Notifications.Discord.BotToken != "" {
			if err := discordService.AuthenticateWithBotToken(config.Notifications.Discord.BotToken); err != nil {
				return nil, fmt.Errorf("discord bot token setup: %w", err)
			}
		} else if config.Notifications.Discord.OAuthToken != "" {
			if err := discordService.AuthenticateWithOAuth2Token(config.Notifications.Discord.OAuthToken); err != nil {
				return nil, fmt.Errorf("discord oauth token setup: %w", err)
			}
		} else {
			panic("logic error")
		}
		discordService.AddReceivers(config.Notifications.Discord.ChannelIDs...)
		services = append(services, discordService)
	}

	if config.Notifications.Email.Server != "" {
		logger.Info("Notifications: using email")
		mailHost := net.JoinHostPort(config.Notifications.Email.Server, strconv.Itoa(config.Notifications.Email.Port))
		mailService := mail.New(config.Notifications.Email.Sender, mailHost)
		if config.Notifications.Email.Username != "" && config.Notifications.Email.Password != "" {
			mailService.AuthenticateSMTP(
				"",
				config.Notifications.Email.Username,
				config.Notifications.Email.Password,
				config.Notifications.Email.Server,
			)
		}
		mailService.AddReceivers(config.Notifications.Email.Recipients...)
		services = append(services, mailService)
	}

	if config.Notifications.SendGrid.APIKey != "" {
		logger.Info("Notifications: using sendgrid")
		sendGridService := sendgrid.New(
			config.Notifications.SendGrid.APIKey,
			config.Notifications.SendGrid.SenderAddress,
			config.Notifications.SendGrid.SenderName,
		)
		sendGridService.AddReceivers(config.Notifications.SendGrid.Recipients...)
		services = append(services, sendGridService)
	}

	if config.Notifications.MSTeams.Webhooks != nil && len(config.Notifications.MSTeams.Webhooks) > 0 {
		logger.Info("Notifications: using msteams")
		msteamsService := msteams.New()
		msteamsService.AddReceivers(config.Notifications.MSTeams.Webhooks...)
		services = append(services, msteamsService)
	}

	not.UseServices(services...)
	return not, nil
}
