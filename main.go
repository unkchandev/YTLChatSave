package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"save-youtube-live-chat/sites"
	"time"

	log "github.com/Sirupsen/logrus"
)

var logger = log.New()
var output = log.New()

const filenameFormat = "2006-01-02 15-04-05"

type YoutubeChatFormatter struct {
}

func (f *YoutubeChatFormatter) Format(entry *log.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	b.WriteString(entry.Message)
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func main() {
	ys := sites.NewYoutubeService()
	if err := os.Mkdir(ys.GetChannelTitle(), 0777); err != nil {
		logger.Debug(err)
	}
	filename := fmt.Sprint(ys.GetChannelTitle(), "\\", time.Now().Format(filenameFormat), ".txt")
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		logger.Debug(err)
	}
	defer f.Close()
	output.Out = f
	output.Formatter = new(YoutubeChatFormatter)

	q := make(chan sites.LiveChatsStr, 2)

	go func() {
		defer close(q)
		var interval int
		for {
			chats, err := ys.GetLiveChats()
			if err != nil {
				logger.Error(err)
			}
			q <- chats
			if chats.PollingIntervalMillis == 0 {
				interval = 500
			} else {
				interval = chats.PollingIntervalMillis
			}
			json.NewEncoder(os.Stdout).Encode(chats)
			time.Sleep(time.Duration(interval) * time.Millisecond)
		}
	}()

	for {
		select {
		case chats := <-q:
			logger.Debug(chats)
			json.NewEncoder(os.Stdout).Encode(chats)
			for _, v := range chats.Items {
				output.Println(
					v.Snippet.PublishedAt.Unix(),
					"\t",
					v.Snippet.DisplayMessage,
				)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
}
