package parser

import (
	"dnsactioneer/model"
	"dnsactioneer/utils"
	"encoding/json"
	"github.com/rs/zerolog/log"
)

func GetConfiguration() (model.WorkConfiguration, error) {
	configString := utils.GetEnvConfig()

	var workConfiguration model.WorkConfiguration
	err := json.Unmarshal([]byte(configString), &workConfiguration)
	if err != nil {
		log.Info().Msgf("Error unmarshalling configuration")
		return model.WorkConfiguration{}, err
	}

	return workConfiguration, nil
}
