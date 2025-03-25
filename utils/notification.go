package utils

import (
	"github.com/gtuk/discordwebhook"
	"github.com/rs/zerolog/log"
)

func SendNotification(messageText string) {
	url := GetEnvDiscordWebhookUrl()

	var username = "DNS Actioneer"

	message := discordwebhook.Message{
		Username: &username,
		Content:  &messageText,
	}

	err := discordwebhook.SendMessage(url, message)
	if err != nil {
		log.Info().Msgf("Error sending notification: %v", err)
	}
}
