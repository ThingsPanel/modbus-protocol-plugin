package tpconfig

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

type SubDeviceFormConfig struct {
	SlaveID        uint8
	CommandRawList []*CommandRaw
}

func NewSubDeviceFormConfig(formConfigMap map[string]interface{}) (*SubDeviceFormConfig, error) {
	// SlaveID
	slaveIDFloat, ok := formConfigMap["SlaveID"].(float64)
	if !ok {
		logrus.Error("SlaveID is not of type float64")
		return nil, fmt.Errorf("SlaveID is not of type float64")
	}

	// CommandRawList
	commandRawListInterface, ok := formConfigMap["CommandRawList"].([]interface{})
	if !ok {
		logrus.Error("CommandRawList is not of type []interface{}")
		return nil, fmt.Errorf("CommandRawList is not of type []interface{}")
	}

	var commandRawList []*CommandRaw
	for _, commandRawMapInterface := range commandRawListInterface {
		commandRawMap, ok := commandRawMapInterface.(map[string]interface{})
		if !ok {
			logrus.Error("commandRawMapInterface is not of type map[string]interface{}")
			return nil, fmt.Errorf("commandRawMapInterface is not of type map[string]interface{}")
		}

		commandRaw, err := NewCommandRaw(commandRawMap)
		if err != nil {
			logrus.Error("NewCommandRaw error:", err)
			continue
		}
		commandRawList = append(commandRawList, commandRaw)
	}

	return &SubDeviceFormConfig{
		SlaveID:        uint8(slaveIDFloat),
		CommandRawList: commandRawList,
	}, nil
}
