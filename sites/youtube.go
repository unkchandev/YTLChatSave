package sites

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var logger = log.New()

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
	ApiKey           string
	ChannelID        string
	videoID          string
	activeLiveChatID string

	searchUrl     string
	liveChatIDUrl string
	liveChatUrl   string

	nextPageToken         string
	pollingIntervalMillis int
}

type LiveInfo struct {
	Title        string
	Description  string
	ChannelTitle string
	StartTime    time.Time
}

var liveInfo LiveInfo

const (
	BASE_SEARCH_URL       = "https://www.googleapis.com/youtube/v3/search?part=snippet&eventType=live&fields=pageInfo%2FtotalResults%2Citems%2Fid%2FvideoId%2Citems%2Fsnippet%2Ftitle%2Citems%2Fsnippet%2Fdescription%2Citems%2Fsnippet%2FchannelTitle&type=video&channelId=__channelID__&key=__key__"
	BASE_LIVE_CHAT_ID_URL = "https://www.googleapis.com/youtube/v3/videos?part=liveStreamingDetails&fields=pageInfo%2FtotalResults%2Citems%2FliveStreamingDetails%2FactualStartTime%2Citems%2FliveStreamingDetails%2FactiveLiveChatId&id=__id__&key=__key__"
	BASE_LIVE_CHAT_URL    = "https://www.googleapis.com/youtube/v3/liveChat/messages?part=snippet&hl=ja&maxResults=2000&fields=items%2Fsnippet%2FdisplayMessage%2Citems%2Fsnippet%2FpublishedAt%2Citems%2Fsnippet%2FauthorChannelId%2CnextPageToken%2CpollingIntervalMillis&liveChatId=__liveChatID__&key=__key__"
)

func NewYoutubeService() *YoutubeService {
	ys := YoutubeService{}
	buf, err := ioutil.ReadFile("config.yml")
	if err != nil {
		panic(err)
	}

	var yc YoutubeConfig
	err = yaml.Unmarshal(buf, &yc)
	if err != nil {
		logger.Info("error: %v", err)
	}

	ys.ApiKey = yc.APIKey
	ys.ChannelID = yc.ChannelID
	ys.setConfig()
	return &ys
}

func (ys *YoutubeService) setConfig() {
	videoID, err := ys.GetLiveID()
	if err != nil {
		logger.Info(err.Error())
	}
	if videoID != "" {
		logger.Info("Now live: " + videoID)
	} else {
		logger.Info("no live...")
	}
	ys.videoID = videoID

	liveChatID, err := ys.GetLiveChatID()
	if err != nil {
		logger.Info(err.Error())
	}
	if liveChatID != "" {
		logger.Info("Now live chat ID: " + liveChatID)
	} else {
		logger.Info("Undefined error.")
	}
	ys.activeLiveChatID = liveChatID
}

func (r *YoutubeService) GetLiveID() (watchID string, err error) {
	var url string
	if r.searchUrl == "" {
		url = strings.Replace(BASE_SEARCH_URL, "__channelID__", r.ChannelID, -1)
		url = strings.Replace(url, "__key__", r.ApiKey, -1)
		r.searchUrl = url
	} else {
		url = r.searchUrl
	}

	logger.Info(url)
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

	liveInfo.Title = s.Items[0].Snippet.Title
	liveInfo.Description = s.Items[0].Snippet.Description
	liveInfo.ChannelTitle = s.Items[0].Snippet.ChannelTitle

	return s.Items[0].ID.VideoID, nil
}

func (r *YoutubeService) GetLiveChatID() (activeLiveChatID string, err error) {
	if r.videoID == "" {
		return "", fmt.Errorf("Unable to access videoID property")
	}

	var url string
	if r.liveChatIDUrl == "" {
		url = strings.Replace(BASE_LIVE_CHAT_ID_URL, "__id__", r.videoID, -1)
		url = strings.Replace(url, "__key__", r.ApiKey, -1)
		r.liveChatIDUrl = url
	} else {
		url = r.liveChatIDUrl
	}

	logger.Info(url)
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
		return "", fmt.Errorf("Unable to get activeLiveChatId from active live: video id %s", r.videoID)
	}

	liveInfo.StartTime = s.Items[0].LiveStreamingDetails.ActualStartTime

	return s.Items[0].LiveStreamingDetails.ActiveLiveChatID, nil
}

func (r *YoutubeService) GetLiveChats() (chats LiveChatsStr, err error) {
	if r.activeLiveChatID == "" {
		return LiveChatsStr{}, fmt.Errorf("Unable to access activeLiveChatID property")
	}
	if r.pollingIntervalMillis > 0 {
		time.Sleep(time.Duration(r.pollingIntervalMillis) * time.Millisecond)
	}

	var url string
	if r.liveChatUrl == "" {
		url = strings.Replace(BASE_LIVE_CHAT_URL, "__liveChatID__", r.activeLiveChatID, -1)
		url = strings.Replace(url, "__key__", r.ApiKey, -1)
		r.liveChatUrl = url
	} else if r.nextPageToken != "" {
		url = r.liveChatUrl + "&pageToken=" + r.nextPageToken
		logger.Debug(r.nextPageToken)
	} else {
		url = r.liveChatUrl
	}

	logger.Info(url)
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
	logger.Debug(s)
	if r.nextPageToken != s.NextPageToken {
		r.nextPageToken = s.NextPageToken
	}

	logger.Info(s)
	return s, nil
}
