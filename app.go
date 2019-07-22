package main

import (
	"net/http"
	"github.com/labstack/echo"
	"os"
	"encoding/json"
	"net/url"
	"io/ioutil"
	"fmt"
)

var (
	name            string = os.Getenv("BOT_NAME")
	slackWebhookUrl string = os.Getenv("SLACK_WEBHOOK_URL")
)

type Slack struct {
	Text     string `json:"text"`
	Username string `json:"username"`
}

func home(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}

func auth(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}

func finshAuth(c echo.Context) error {
	return c.String(http.StatusOK, "Completed!")
}

func members(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}

func echoToSlack(c echo.Context) error {
	params, _ := json.Marshal(Slack{
		"Hello",
		name,
	})
	resp, _ := http.PostForm(
		slackWebhookUrl,
		url.Values{"payload": {string(params)}},
	)

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	fmt.Println(string(body))

	return c.String(http.StatusOK, "OK")
}

func main() {
	e := echo.New()
	e.GET("/", home)
	e.GET("/auth", auth)
	e.GET("/ok", finshAuth)
	e.GET("/members", members)
	e.GET("/echo", echoToSlack)

	e.Logger.Fatal(e.Start(":8080"))
}
