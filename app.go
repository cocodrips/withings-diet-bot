package main

import (
	"net/http"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"os"
	"encoding/json"
	"net/url"
	"io/ioutil"
	"fmt"
	"github.com/labstack/echo/middleware"
	"path"
	"strings"
)

var (
	selfUrl         string = os.Getenv("SELF_URL")
	name            string = os.Getenv("BOT_NAME")
	slackWebhookUrl string = os.Getenv("SLACK_WEBHOOK_URL")
	clientId        string = os.Getenv("CLIENT_ID")
	secret          string = os.Getenv("SECRET")
	authUrl         string = "https://account.withings.com/oauth2_user/authorize2"
	tokenUrl        string = "https://account.withings.com/oauth2/token"
	dataFile        string = "diet-token.json"
)

type User struct {
	Name         string `json:"name"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
type Slack struct {
	Text     string `json:"text"`
	Username string `json:"username"`
}

// endpoints

func home(c echo.Context) error {
	return c.String(http.StatusOK, name)
}

func finshAuth(c echo.Context) error {
	//authenticationCode := c.Param("code")
	return c.String(http.StatusOK, "Completed!")
}

func members(c echo.Context) error {
	users := readUser()

	names := []string{}
	for _, user := range users {
		names = append(names, user.Name)
	}
	return c.String(http.StatusOK, strings.Join(names, ","))
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

	return c.String(http.StatusOK, name)
}

func auth(c echo.Context) error {
	u, err := url.Parse(authUrl)
	if err != nil {
		return err
	}

	redirectUrl, _ := url.Parse(selfUrl)
	redirectUrl.Path = path.Join(redirectUrl.Path, "user/add")

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientId)
	q.Set("state", "diet")
	q.Set("scope", "user.metrics")
	q.Set("redirect_uri", redirectUrl.String())
	q.Set("mode", "demo")
	u.RawQuery = q.Encode()

	_, err = http.Get(u.String())
	if err != nil {
		return c.String(http.StatusUnauthorized, "OK")
	}

	return c.HTML(http.StatusOK,
		fmt.Sprintf("<a href='%s'>Authentication</a>", u.String()))
}

func receiveAuthenticationCode(c echo.Context) error {
	code := c.QueryParam("code")
	log.Infof(code)

	u, _ := url.Parse(selfUrl)
	u.Path = path.Join(u.Path, "user/add")

	q := u.Query()
	q.Set("code", code)
	u.RawQuery = q.Encode()

	log.Infof("token url: %s", u.String())

	html := fmt.Sprintf(`
<form method="post" action="%s">
  <label for="diet_name">name:</label>
  <input type="text" name="diet_name"><input type="submit">
</form>`, u.String())
	return c.HTML(http.StatusOK, html)
}

func saveAccessToken(c echo.Context) error {

	code := c.QueryParam("code")
	name := c.FormValue("diet_name")

	resp, err := getAccessToken(code)
	defer resp.Body.Close()

	if err != nil {
		log.Fatal("Failed to get access token")
		c.String(http.StatusOK, "Failed")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	tokenResponse := TokenResponse{}
	err = json.Unmarshal(body, &tokenResponse)
	user := User{
		Name:         name,
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
	}
	addUser(user)

	return c.String(http.StatusOK, "OK")
}

// util

func getAccessToken(code string) (*http.Response, error) {
	values := url.Values{}

	redirectUrl, _ := url.Parse(selfUrl)
	redirectUrl.Path = path.Join(redirectUrl.Path, "user/add")

	values.Add("grant_type", "authorization_code")
	values.Add("client_id", clientId)
	values.Add("client_secret", secret)
	values.Add("code", code)
	values.Add("redirect_uri", redirectUrl.String())

	return http.PostForm(tokenUrl, values)
}

func readUser() []User {
	bytes, err := ioutil.ReadFile(dataFile)
	if err != nil {
		log.Error(err)
	}

	var users []User
	if err := json.Unmarshal(bytes, &users); err != nil {
		log.Error(err)
	}

	return users
}

func addUser(user User) {
	// Add user
	users := readUser()
	users = append(users, user)

	userJson, err := json.MarshalIndent(users, "", "    ")
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile(dataFile, userJson, os.ModePerm)
}

//
func main() {
	e := echo.New()
	e.GET("/", home)
	e.GET("/auth", auth)
	e.GET("/ok", finshAuth)
	e.GET("/members", members)
	e.GET("/echo", echoToSlack)
	e.GET("/user/add", receiveAuthenticationCode)
	e.POST("/user/add", saveAccessToken)

	fp, err := os.OpenFile("/var/log/app/access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	dataFile, err := os.OpenFile(dataFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	dataFile.Close()

	// Logger
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
		Output: fp,
	}))
	log.SetOutput(fp)
	e.Logger.SetLevel(log.INFO)

	e.Use(middleware.Recover())

	e.Logger.Fatal(e.Start(":80"))
}
