package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type LiveCheckStr struct {
	PageInfo struct {
		TotalResults int `json:"totalResults"`
	} `json:"pageInfo"`
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			Title        string `json:"title"`
			Description  string `json:"description"`
			ChannelTitle string `json:"channelTitle"`
		} `json:"snippet"`
	} `json:"items"`
}

type LiveChatIDStr struct {
	PageInfo struct {
		TotalResults int `json:"totalResults"`
	} `json:"pageInfo"`
	Items []struct {
		LiveStreamingDetails struct {
			ActualStartTime  time.Time `json:"actualStartTime"`
			ActiveLiveChatID string    `json:"activeLiveChatId"`
		} `json:"liveStreamingDetails"`
	} `json:"items"`
}

type LiveChatsStr struct {
	NextPageToken         string `json:"nextPageToken"`
	PollingIntervalMillis int    `json:"pollingIntervalMillis"`
	Items                 []struct {
		Snippet struct {
			AuthorChannelID string    `json:"authorChannelId"`
			PublishedAt     time.Time `json:"publishedAt"`
			DisplayMessage  string    `json:"displayMessage"`
		} `json:"snippet"`
	} `json:"items"`
}

type YoutubeConfig struct {
	APIKey    string `yaml:"APIKey"`
	ChannelID string `yaml:"ChannelID"`
}

type YoutubeService struct {
	config YoutubeConfig
	info   liveInfo

	videoID          string
	activeLiveChatID string

	searchUrl     string
	liveChatIDUrl string
	liveChatUrl   string

	nextPageToken string

	logch chan string
}

type liveInfo struct {
	title        string
	description  string
	channelTitle string
	startTime    time.Time
}

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
	b.WriteByte('\r')
	b.WriteByte('\n')
	return b.Bytes(), nil
}

const (
	BASE_SEARCH_URL       = "https://www.googleapis.com/youtube/v3/search?part=snippet&eventType=live&fields=pageInfo%2FtotalResults%2Citems%2Fid%2FvideoId%2Citems%2Fsnippet%2Ftitle%2Citems%2Fsnippet%2Fdescription%2Citems%2Fsnippet%2FchannelTitle&type=video&channelId=__channelID__&key=__key__"
	BASE_LIVE_CHAT_ID_URL = "https://www.googleapis.com/youtube/v3/videos?part=liveStreamingDetails&fields=pageInfo%2FtotalResults%2Citems%2FliveStreamingDetails%2FactualStartTime%2Citems%2FliveStreamingDetails%2FactiveLiveChatId&id=__id__&key=__key__"
	BASE_LIVE_CHAT_URL    = "https://www.googleapis.com/youtube/v3/liveChat/messages?part=snippet&hl=ja&maxResults=2000&fields=items%2Fsnippet%2FdisplayMessage%2Citems%2Fsnippet%2FpublishedAt%2Citems%2Fsnippet%2FauthorChannelId%2CnextPageToken%2CpollingIntervalMillis&liveChatId=__liveChatID__&key=__key__"
	CONFIG_FILE           = "config.yml"
)

func NewYoutubeService(logch chan string) (*YoutubeService, error) {
	ys := YoutubeService{}
	ys.logch = logch

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

	ys.config = yc
	return &ys, nil
}

func (ys *YoutubeService) Init() {
	ys.videoID = ""
	ys.activeLiveChatID = ""
	ys.liveChatIDUrl = ""
	ys.liveChatUrl = ""
	ys.info = liveInfo{}
}

func (ys *YoutubeService) GetChannelTitle() string {
	return ys.info.channelTitle
}

func (ys *YoutubeService) SetConfig() error {
	for {
		videoID, err := ys.getLiveID()
		if err != nil || videoID == "" {
			time.Sleep(5 * time.Second)
			continue
		}
		ys.logch <- "Live detected! video id: " + videoID
		ys.videoID = videoID

		liveChatID, err := ys.getLiveChatID()
		if err != nil || liveChatID == "" {
			time.Sleep(5 * time.Second)
			continue
		}
		ys.logch <- "Live chat detected! chat id: " + liveChatID

		ys.activeLiveChatID = liveChatID
		return nil
	}
}

func (ys *YoutubeService) CheckLive() (isLive bool, err error) {
	videoID, err := ys.getLiveID()
	if err != nil {
		return false, err
	}
	if videoID == "" {
		return false, nil
	}
	return true, nil

}

func (ys *YoutubeService) getLiveID() (string, error) {
	var url string
	if ys.searchUrl == "" {
		url = strings.Replace(BASE_SEARCH_URL, "__channelID__", ys.config.ChannelID, -1)
		url = strings.Replace(url, "__key__", ys.config.APIKey, -1)
		ys.searchUrl = url
	} else {
		url = ys.searchUrl
	}

	//get
	res, err := http.Get(url)
	if err != nil {
		return "", err
	} else if res.StatusCode != 200 {
		return "", fmt.Errorf("Unable to get this url : http status %d", res.StatusCode)
	}
	defer res.Body.Close()

	//read body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	//decode
	var s LiveCheckStr
	if err := json.Unmarshal(body, &s); err != nil {
		return "", err
	}

	//check
	if s.PageInfo.TotalResults == 0 {
		return "", nil
	}

	ys.info.title = s.Items[0].Snippet.Title
	ys.info.description = s.Items[0].Snippet.Description
	ys.info.channelTitle = s.Items[0].Snippet.ChannelTitle
	time.Sleep(5 * time.Second)

	return s.Items[0].ID.VideoID, nil
}

func (ys *YoutubeService) getLiveChatID() (activeLiveChatID string, err error) {
	if ys.videoID == "" {
		return "", fmt.Errorf("Unable to access videoID property.")
	}

	var url string
	url = strings.Replace(BASE_LIVE_CHAT_ID_URL, "__id__", ys.videoID, -1)
	url = strings.Replace(url, "__key__", ys.config.APIKey, -1)
	ys.liveChatIDUrl = url

	//get
	res, err := http.Get(url)
	if err != nil {
		return "", err
	} else if res.StatusCode != 200 {

		return "", fmt.Errorf("Unable to get this url : http status %d", res.StatusCode)
	}
	defer res.Body.Close()

	//read body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	//decode
	var s LiveChatIDStr
	if err := json.Unmarshal(body, &s); err != nil {
		return "", err
	}

	//check
	if s.PageInfo.TotalResults == 0 {
		//waiting api
		return "", nil
	}

	ys.info.startTime = s.Items[0].LiveStreamingDetails.ActualStartTime
	time.Sleep(5 * time.Second)

	return s.Items[0].LiveStreamingDetails.ActiveLiveChatID, nil
}

func (ys *YoutubeService) GetLiveChats() (chats LiveChatsStr, err error) {
	if ys.activeLiveChatID == "" {
		return LiveChatsStr{}, fmt.Errorf("Unable to access activeLiveChatID property.")
	}

	var url string
	if ys.liveChatUrl == "" {
		url = strings.Replace(BASE_LIVE_CHAT_URL, "__liveChatID__", ys.activeLiveChatID, -1)
		url = strings.Replace(url, "__key__", ys.config.APIKey, -1)
		ys.liveChatUrl = url
	}
	if ys.nextPageToken != "" {
		url = ys.liveChatUrl + "&pageToken=" + ys.nextPageToken
	}

	//get
	res, _ := http.Get(url)
	if err != nil {
		return LiveChatsStr{}, err
	} else if res.StatusCode != 200 {
		return LiveChatsStr{}, fmt.Errorf("Unable to get this url : http status %d", res.StatusCode)
	}
	defer res.Body.Close()

	//read body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return LiveChatsStr{}, err
	}

	//decode
	var s LiveChatsStr
	if err := json.Unmarshal(body, &s); err != nil {
		return LiveChatsStr{}, err
	}

	//check
	if ys.nextPageToken != s.NextPageToken {
		ys.nextPageToken = s.NextPageToken
	}

	return s, nil
}
