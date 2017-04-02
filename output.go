package main

import (
	_ "fmt"

	log "github.com/Sirupsen/logrus"
)

type YoutubeChatFormatter struct {
}

var logger = log.New()

//func (f *YoutubeChatFormatter) Format(entry *Entry) (byte[], error) {
//serialized, err := json.Marshal(entry.Data)
//if err != nil {
//	return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
//}
//return append(serialized, '\n'), nil
//	return nil, nil
//}

func GetLogger() *log.Logger {
	return logger
}
