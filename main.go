package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"

	log "github.com/Sirupsen/logrus"
)

var output = log.New()
var logch = make(chan string, 10)
var livech = make(chan bool, 2)

const (
	FILENAME_FORMAT = "2006-01-02 15-04-05"
	LOG_FORMAT      = "2006-01-02 15:04:05"
	LOCATION        = "Asia/Tokyo"
)

func main() {
	setLocation()
	go mainLoop()
	go logging()
	output.Formatter = new(YoutubeChatFormatter)

	runMainWindow()
}

func mainLoop() {
	// read config
	buf, err := ioutil.ReadFile(CONFIG_FILE)
	if err != nil {
		logch <- "Unable to read config file. Message: " + err.Error()
		return
	}

	var yc YoutubeConfig
	err = yaml.Unmarshal(buf, &yc)
	if err != nil {
		logch <- "Unable to parse config file. Message: " + err.Error()
		return
	}

	ys, err := NewYoutubeService(yc, logch)
	if err != nil {
		logch <- "Unable to load config.yml file. Error:" + err.Error()
		return
	}
	logch <- "Load completed. Start monitoring."
	go checkLiveLoop(ys)
}

func checkLiveLoop(ys *YoutubeService) {
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
			logch <- "Live finished."
			ys.Init()
			ys.nextPageToken = ""
			livech <- false
		}
		time.Sleep(5 * time.Second)
	}
}

func ChatSave(ys *YoutubeService) {
	ys.SetConfig()
	err := createLiveInfoFile(ys)
	if err != nil {
		logch <- err.Error()
	}

	f, err := openChatsFile(ys)
	if err != nil {
		logch <- err.Error()
	}
	defer f.Close()
	output.Out = f

	var chatsNum = 0
	for {
		select {
		case isLive := <-livech:
			if !isLive {
				return
			}
		default:
			var interval int
			chats, err := ys.GetLiveChats()
			if err != nil {
				logch <- err.Error()
				continue
			}
			if chats.PollingIntervalMillis == 0 {
				interval = 5000
			} else {
				interval = chats.PollingIntervalMillis
			}
			time.Sleep(time.Duration(interval) * time.Millisecond)

			if len(chats.Items) == 0 && chatsNum == 0 {
				ys.Init()
				ys.SetConfig()
				continue
			}
			chatsNum = len(chats.Items)

			//output to file
			for _, v := range chats.Items {
				text := strconv.FormatInt(v.Snippet.PublishedAt.Unix(), 10) +
					"\t" +
					v.Snippet.DisplayMessage
				output.Println(text)
			}
		}
	}
}

func createLiveInfoFile(ys *YoutubeService) error {
	if _, err := os.Stat(ys.GetChannelTitle()); os.IsNotExist(err) {
		if err := os.Mkdir(ys.GetChannelTitle(), 0777); err != nil {
			return err
		}
	}

	filename := fmt.Sprint(ys.GetChannelTitle(), "\\",
		time.Now().Format(FILENAME_FORMAT), "-info", ".txt")
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return err
	}
	defer f.Close()

	var info liveInfo
	for {
		info, err = ys.GetLiveInfo()
		if err != nil {
			logch <- err.Error()
		}
		if info.channelTitle == "" || info.description == "" || info.title == "" {
			continue
		}
		break
	}
	text := fmt.Sprint(
		"Channel Title: ", info.channelTitle, "\r\n",
		"Live Title: ", info.title, "\r\n",
		"Video ID: ", ys.GetVideoID(), "\r\n",
		"Description: ", info.description, "\r\n",
		"Start time: ", info.startTime.In(time.Local).Format(LOG_FORMAT), "\r\n",
	)
	f.WriteString(text)
	return nil
}

func openChatsFile(ys *YoutubeService) (*os.File, error) {
	if _, err := os.Stat(ys.GetChannelTitle()); os.IsNotExist(err) {
		if err := os.Mkdir(ys.GetChannelTitle(), 0777); err != nil {
			return nil, err
		}
	}
	filename := fmt.Sprint(ys.GetChannelTitle(), "\\",
		time.Now().Format(FILENAME_FORMAT), ".txt")
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func loadConfig() (*YoutubeConfig, error) {
	// read config
	buf, err := ioutil.ReadFile(CONFIG_FILE)
	if err != nil {
		return nil, fmt.Errorf("Unable to read config file. Message: " + err.Error())
	}

	var yc YoutubeConfig
	err = yaml.Unmarshal(buf, &yc)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse config file. Message: " + err.Error())
	}
	return &yc, nil
}

func saveConfigFile(yc *YoutubeConfig) error {
	d, err := yaml.Marshal(yc)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(CONFIG_FILE, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString(string(d))
	return nil
}

func setLocation() {
	loc, err := time.LoadLocation(LOCATION)
	if err != nil {
		loc = time.FixedZone(LOCATION, 9*60*60)
	}
	time.Local = loc
}

func logging() {
	for mw.logTE == nil {
		time.Sleep(100 * time.Millisecond)
	}
	for {
		msg := <-logch
		mw.logTE.AppendText(time.Now().Format(LOG_FORMAT) + "[LOG]" + msg + "\r\n")
	}
}
