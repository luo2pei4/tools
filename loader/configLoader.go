package loader

import (
	"errors"
	"strings"

	"github.com/pelletier/go-toml"
)

var configuration *toml.Tree

// LoadConfig 加载配置文件
func LoadConfig(path string) (success bool, err error) {

	config, err := toml.LoadFile(path)
	configuration = config

	if err != nil {

		return false, errors.New("Load config file failed")
	}

	return true, nil
}

// GetTable 获取配置信息的表
func GetTable(key string) (table *toml.Tree, err error) {

	if configuration == nil {

		return nil, errors.New("config info was not loaded, call LoadConfig first")
	}

	if strings.TrimSpace(key) == "" {

		return nil, errors.New("key is invariable")
	}

	if t, ok := configuration.Get(key).(*toml.Tree); ok {

		return t, nil
	}

	return nil, errors.New("Cannot get table, check table name again")
}

// GetValue 获取配置文件的数据
func GetValue(key string) (value interface{}, err error) {

	if configuration == nil {

		return nil, errors.New("config info was not loaded")
	}

	if strings.TrimSpace(key) == "" {

		return nil, errors.New("key is blank")
	}

	value = configuration.Get(key)

	return
}

// IntValue get int value
func IntValue(config *toml.Tree, key string, defaultValue int) int {

	v := config.Get(key)

	if v != nil {

		if value, ok := v.(int); ok {

			return value
		}
	}

	return defaultValue
}

// StringValue get string value
func StringValue(config *toml.Tree, key string, defaultValue string) string {

	v := config.Get(key)

	if v != nil {

		if value, ok := v.(string); ok {

			return value
		}
	}

	return defaultValue
}

// BoolValue get boolean value
func BoolValue(config *toml.Tree, key string, defaultValue bool) bool {

	v := config.Get(key)

	if v != nil {

		if value, ok := v.(bool); ok {

			return value
		}
	}

	return defaultValue
}
