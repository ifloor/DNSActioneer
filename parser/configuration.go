package parser

import (
	"dnsactioneer/model"
	"dnsactioneer/utils"
	"encoding/json"
	"log"
)

func GetConfiguration() (model.WorkConfiguration, error) {
	configString := utils.GetEnvConfig()

	var workConfiguration model.WorkConfiguration
	err := json.Unmarshal([]byte(configString), &workConfiguration)
	if err != nil {
		log.Println("Error unmarshalling configuration")
		return model.WorkConfiguration{}, err
	}

	return workConfiguration, nil
}
