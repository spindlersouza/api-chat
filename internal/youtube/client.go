package youtube

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"api-chat/internal/chatmsg"
)

const apiBase = "https://www.googleapis.com/youtube/v3"

// Run polls YouTube Live Chat for the given channel and streams messages
// into onMessage. It rediscovers the live video whenever the chat ends,
// and retries with backoff on any error.
func Run(apiKey, channelID string, minIntervalMs int, onMessage func(chatmsg.Message)) {
	if apiKey == "" || channelID == "" {
		log.Println("youtube: YOUTUBE_API_KEY or YOUTUBE_CHANNEL_ID not set, skipping")
		return
	}

	for {
		liveChatID, err := findActiveLiveChatID(apiKey, channelID)
		if err != nil {
			log.Println("youtube: could not find active live:", err)
			time.Sleep(60 * time.Second)
			continue
		}
		if liveChatID == "" {
			log.Println("youtube: no active live stream found, retrying in 60s")
			time.Sleep(60 * time.Second)
			continue
		}

		log.Println("youtube: connected to live chat")
		err = pollLiveChat(apiKey, liveChatID, minIntervalMs, onMessage)
		if err != nil {
			log.Println("youtube: polling stopped:", err)
		}
		time.Sleep(5 * time.Second)
	}
}

type searchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
	} `json:"items"`
}

type videoResponse struct {
	Items []struct {
		LiveStreamingDetails struct {
			ActiveLiveChatID string `json:"activeLiveChatId"`
		} `json:"liveStreamingDetails"`
	} `json:"items"`
}

func findActiveLiveChatID(apiKey, channelID string) (string, error) {
	searchURL := fmt.Sprintf("%s/search?part=snippet&channelId=%s&eventType=live&type=video&key=%s",
		apiBase, url.QueryEscape(channelID), url.QueryEscape(apiKey))

	var sr searchResponse
	if err := getJSON(searchURL, &sr); err != nil {
		return "", err
	}
	if len(sr.Items) == 0 {
		return "", nil
	}
	videoID := sr.Items[0].ID.VideoID

	videoURL := fmt.Sprintf("%s/videos?part=liveStreamingDetails&id=%s&key=%s",
		apiBase, url.QueryEscape(videoID), url.QueryEscape(apiKey))

	var vr videoResponse
	if err := getJSON(videoURL, &vr); err != nil {
		return "", err
	}
	if len(vr.Items) == 0 {
		return "", nil
	}
	return vr.Items[0].LiveStreamingDetails.ActiveLiveChatID, nil
}

type liveChatResponse struct {
	NextPageToken         string `json:"nextPageToken"`
	PollingIntervalMillis int    `json:"pollingIntervalMillis"`
	Items                 []struct {
		Snippet struct {
			DisplayMessage string `json:"displayMessage"`
			PublishedAt    string `json:"publishedAt"`
		} `json:"snippet"`
		AuthorDetails struct {
			DisplayName     string `json:"displayName"`
			ProfileImageURL string `json:"profileImageUrl"`
			IsChatOwner     bool   `json:"isChatOwner"`
			IsChatModerator bool   `json:"isChatModerator"`
			IsChatSponsor   bool   `json:"isChatSponsor"`
		} `json:"authorDetails"`
	} `json:"items"`
}

func pollLiveChat(apiKey, liveChatID string, minIntervalMs int, onMessage func(chatmsg.Message)) error {
	pageToken := ""

	for {
		endpoint := fmt.Sprintf("%s/liveChat/messages?liveChatId=%s&part=snippet,authorDetails&key=%s",
			apiBase, url.QueryEscape(liveChatID), url.QueryEscape(apiKey))
		if pageToken != "" {
			endpoint += "&pageToken=" + url.QueryEscape(pageToken)
		}

		var resp liveChatResponse
		if err := getJSON(endpoint, &resp); err != nil {
			return err
		}

		for _, item := range resp.Items {
			badges := []string{}
			if item.AuthorDetails.IsChatOwner {
				badges = append(badges, "owner")
			}
			if item.AuthorDetails.IsChatModerator {
				badges = append(badges, "moderator")
			}
			if item.AuthorDetails.IsChatSponsor {
				badges = append(badges, "member")
			}

			ts := time.Now().UnixMilli()
			if parsed, err := time.Parse(time.RFC3339, item.Snippet.PublishedAt); err == nil {
				ts = parsed.UnixMilli()
			}

			onMessage(chatmsg.Message{
				Platform:  "youtube",
				Username:  item.AuthorDetails.DisplayName,
				Message:   item.Snippet.DisplayMessage,
				Avatar:    item.AuthorDetails.ProfileImageURL,
				Badges:    badges,
				Color:     "",
				Emotes:    []chatmsg.Emote{},
				Timestamp: ts,
			})
		}

		pageToken = resp.NextPageToken

		// Ignora o pollingIntervalMillis sugerido pela API e usa sempre o
		// intervalo fixo configurado (mais rapido, consome cota mais rapido).
		time.Sleep(time.Duration(minIntervalMs) * time.Millisecond)
	}
}

func getJSON(endpoint string, out interface{}) error {
	resp, err := http.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("youtube api error %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, out)
}
