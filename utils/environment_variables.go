package utils

import (
	"log"
	"os"
	"strconv"
)

const DefaultLoopIntervalSeconds = 300

func GetEnvConfig() string {
	config := os.Getenv("CONFIG")
	if config == "" {
		log.Println("CONFIG environment variable is not set")
	}
	return config
}

func GetEnvLoopIntervalSeconds() int {
	loopIntervalSecondsStr := os.Getenv("LOOP_INTERVAL_SECONDS")
	if loopIntervalSecondsStr == "" {
		log.Println("LOOP_INTERVAL_SECONDS environment variable is not set")

		return DefaultLoopIntervalSeconds
	}

	loopIntervalSeconds, err := strconv.Atoi(loopIntervalSecondsStr)
	if err != nil {
		log.Println("Error converting LOOP_INTERVAL_SECONDS to int")
		return DefaultLoopIntervalSeconds
	}

	return loopIntervalSeconds
}

func GetEnvDOToken() string {
	doToken := os.Getenv("DO_TOKEN")
	if doToken == "" {
		log.Println("DO_TOKEN environment variable is not set")
	}
	return doToken
}

func GetEnvDiscordWebhookUrl() string {
	url := os.Getenv("DISCORD_WH_URL")
	if url == "" {
		log.Println("DISCORD_WH_URL environment variable is not set")
		panic("DISCORD_WH_URL environment variable is not set")
	}
	return url
}
