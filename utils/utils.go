package utils

import (
	"encoding/json"
	"strings"
)

func FormatGitRepoName(slug string) string {
	name := strings.Replace(slug, "/", "-", -1)
	name = strings.Replace(name, "_", "-", -1)
	name = strings.Replace(name, " ", "-", -1)
	return name
}

func EnJson(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

func DeJson(jsondata string, data interface{}) error {
	err := json.Unmarshal([]byte(jsondata), data)
	if err != nil {
		return err
	}
	return nil
}

func StringInArray(value string, array []string) bool {
	DataMap := make(map[string]bool, 0)
	for _, item := range array {
		DataMap[item] = true
	}
	_, ok := DataMap[value]
	return ok
}

func Int64InArray(value int64, array []int64) bool {
	DataMap := make(map[int64]bool, 0)
	for _, item := range array {
		DataMap[item] = true
	}
	_, ok := DataMap[value]
	return ok
}

func IntInArray(value int, array []int) bool {
	DataMap := make(map[int]bool, 0)
	for _, item := range array {
		DataMap[item] = true
	}
	_, ok := DataMap[value]
	return ok
}
