package main

import (
	"fmt"

	"yt_chat/sites"
)

const (
	CHAT_INTERVAL_MS = 500
)

func main() {
	ys := sites.NewYoutubeService()
	fmt.Println(ys.GetLiveChats())
}
