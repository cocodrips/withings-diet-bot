package slack

import (
	"os"
	"encoding/json"
	"net/http"
	"net/url"
	"io/ioutil"
	"fmt"
)

var (
	botName         string = os.Getenv("BOT_NAME")
	slackWebhookUrl string = os.Getenv("SLACK_WEBHOOK_URL")
)

type Slack struct {
	Text     string `json:"text"`
	Username string `json:"username"`
}

func EchoToSlack(message string) {
	params, _ := json.Marshal(Slack{
		message,
		botName,
	})
	resp, _ := http.PostForm(
		slackWebhookUrl,
		url.Values{"payload": {string(params)}},
	)

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	fmt.Println(string(body))
}
