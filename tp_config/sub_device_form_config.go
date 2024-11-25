package tpconfig

import (
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
)

type SubDeviceFormConfig struct {
	SlaveID        uint8
	CommandRawList []*CommandRaw
}

func NewSubDeviceFormConfig(formConfigMap map[string]interface{}, subDeviceAddr string) (*SubDeviceFormConfig, error) {
	// SlaveID
	var slaveIDFloat float64
	slaveIDInterface, exists := formConfigMap["SlaveID"]
	if !exists {
		// 如果 SlaveID 不存在，使用 subDeviceAddr
		s, err := strconv.ParseFloat(subDeviceAddr, 64)
		if err != nil {
			logrus.Error("子设备地址不是有效的数字格式:", err)
			return nil, fmt.Errorf("子设备地址必须是有效的数字格式")
		}
		slaveIDFloat = s
	} else {
		// 尝试从 formConfigMap 中获取 SlaveID
		if floatVal, ok := slaveIDInterface.(float64); ok {
			slaveIDFloat = floatVal
		} else if strVal, ok := slaveIDInterface.(string); ok {
			// 如果是字符串，尝试转换为数字
			s, err := strconv.ParseFloat(strVal, 64)
			if err != nil {
				logrus.Error("从配置中获取的从站ID不是有效的数字格式:", err)
				return nil, fmt.Errorf("从站ID必须是有效的数字格式")
			}
			slaveIDFloat = s
		} else {
			logrus.Error("从站ID格式无效")
			return nil, fmt.Errorf("从站ID格式无效")
		}
	}

	// CommandRawList
	commandRawListInterface, ok := formConfigMap["CommandRawList"].([]interface{})
	if !ok {
		logrus.Error("命令列表格式无效")
		return nil, fmt.Errorf("命令列表格式无效")
	}

	var commandRawList []*CommandRaw
	for _, commandRawMapInterface := range commandRawListInterface {
		commandRawMap, ok := commandRawMapInterface.(map[string]interface{})
		if !ok {
			logrus.Error("命令项格式无效")
			return nil, fmt.Errorf("命令项格式无效")
		}

		commandRaw, err := NewCommandRaw(commandRawMap)
		if err != nil {
			logrus.Error("创建命令对象失败:", err)
			continue
		}
		commandRawList = append(commandRawList, commandRaw)
	}

	return &SubDeviceFormConfig{
		SlaveID:        uint8(slaveIDFloat),
		CommandRawList: commandRawList,
	}, nil
}
