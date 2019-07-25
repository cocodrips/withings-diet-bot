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
	clientId        string = os.Getenv("CLIENT_ID")
	secret          string = os.Getenv("SECRET")
	authUrl         string = "https://account.withings.com/oauth2_user/authorize2"
)

type Slack struct {
	Text     string `json:"text"`
	Username string `json:"username"`
}

func home(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}

func auth(c echo.Context) error {
	// endpoint のスキームでミスがあったら、ここでチェックできる。
	//u, err := url.Parse(authUrl)
	//if err != nil {
	//	return err
	//}

	//// 元のURLにクエリパラメータがついていた場合の上書きを防げる。
	//q := u.Query()
	//q.Set("response_type", "code")
	//q.Set("client_id", clientId)
	//q.Set("scope", "user.metrics")
	//u.RawQuery = q.Encode()
	//
	//req, err := http.Get(u.String())
	return c.String(http.StatusOK, "OK")
}

func finshAuth(c echo.Context) error {
	//authenticationCode := c.Param("code")
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

func authCallBack(c echo.Context) error {
	code := c.QueryParam("code")
	state := c.QueryParam("state")

	message := fmt.Sprintf("code:%s\nstate:%s", code, state)
	return c.String(http.StatusOK, message)
}

func main() {
	e := echo.New()
	e.GET("/", home)
	e.GET("/auth", auth)
	e.GET("/ok", finshAuth)
	e.GET("/members", members)
	e.GET("/echo", echoToSlack)
	e.GET("/callback", authCallBack)


	e.Logger.Fatal(e.Start(":80"))
}
