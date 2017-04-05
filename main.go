package main

import (
	"fmt"
	"os"
	"save-youtube-live-chat/sites"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
)

type LogWindow struct {
	*walk.MainWindow

	logTE *walk.TextEdit
}

var output = log.New()
var mw = &LogWindow{}
var logch = make(chan string, 10)
var livech = make(chan bool, 2)

const filenameFormat = "2006-01-02 15-04-05"

func main() {
	go mainLoop()
	go logging()
	output.Formatter = new(sites.YoutubeChatFormatter)

	MW := MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "save youtube live chat",
		MinSize:  Size{600, 450},
		Layout:   VBox{},
		Children: []Widget{
			TextEdit{AssignTo: &mw.logTE, ReadOnly: true},
		},
		MenuItems: []MenuItem{
			Menu{
				Text: "&File",
				Items: []MenuItem{
					Separator{},
					Action{
						Text:        "Exit",
						OnTriggered: func() { mw.Close() },
					},
				},
			},
		},
	}
	MW.Run()
}

func mainLoop() {
	ys, err := sites.NewYoutubeService(logch)
	if err != nil {
		logch <- "Unable to load config.yml file. Error:" + err.Error()
		return
	}
	logch <- "Load completed. Start monitoring."
	go checkLiveLoop(ys)
}

func checkLiveLoop(ys *sites.YoutubeService) {
	isLiveOld := false
	for {
		isLive, err := ys.CheckLive()
		if err != nil {
			output.Println(err)
		}
		if isLive && !isLiveOld {
			isLiveOld = true
			logch <- "Live start on " + ys.GetChannelTitle() + ". Start saving chats log."
			go ChatSave(ys)
		} else if !isLive && isLiveOld {
			isLiveOld = false
			logch <- "Live finished on " + ys.GetChannelTitle()
			ys.Init()
			livech <- false
		}
		time.Sleep(5 * time.Second)
	}
}

func ChatSave(ys *sites.YoutubeService) {
	q := make(chan sites.LiveChatsStr, 5)
	endch := make(chan bool, 2)

	if _, err := os.Stat(ys.GetChannelTitle()); os.IsNotExist(err) {
		if err := os.Mkdir(ys.GetChannelTitle(), 0777); err != nil {
			logch <- err.Error()
		}
	}

	filename := fmt.Sprint(ys.GetChannelTitle(), "\\", time.Now().Format(filenameFormat), ".txt")
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		logch <- err.Error()
	}
	defer f.Close()
	output.Out = f

	ys.SetConfig()

	go func(livech chan bool) {
		var interval int
		for {
			chats, err := ys.GetLiveChats()
			if err != nil {
				logch <- err.Error()
			}
			select {
			case q <- chats:
				if chats.PollingIntervalMillis == 0 {
					interval = 5000
				} else {
					interval = chats.PollingIntervalMillis
				}
				time.Sleep(time.Duration(interval) * time.Millisecond)
			case isLive := <-livech:
				if !isLive {
					return
				}
			}
		}
	}(endch)

	for {
		select {
		case chats := <-q:
			for _, v := range chats.Items {
				text := strconv.FormatInt(v.Snippet.PublishedAt.Unix(), 10) +
					"\t" +
					v.Snippet.DisplayMessage
				output.Println(text)
			}
		case isLive := <-livech:
			endch <- false
			if !isLive {
				return
			}
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func logging() {
	for {
		msg := <-logch
		mw.logTE.AppendText(time.Now().Format("2006-01-02 15:04:05") + "[LOG]" + msg + "\n")
	}
}
