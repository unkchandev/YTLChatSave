package main

import (
	"encoding/json"
	"os"
	"save-youtube-live-chat/sites"
	"time"
)

func main() {
	ys := sites.NewYoutubeService()
	logger := GetLogger()

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
		}
		time.Sleep(10 * time.Millisecond)
	}
}
