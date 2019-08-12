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
	"time"
	"math"
	"sort"
)

var (
	name            string = os.Getenv("BOT_NAME")
	slackWebhookUrl string = os.Getenv("SLACK_WEBHOOK_URL")
	clientId        string = os.Getenv("CLIENT_ID")
	secret          string = os.Getenv("SECRET")
	dataFile        string = "diet-token.json"
	selfUrl         *url.URL
	authUrl         *url.URL
	tokenUrl        *url.URL
	measureUrl      *url.URL
)

type User struct {
	Name         string `json:"name"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type UserStatus struct {
	Name  string
	Ratio float64
}

// ------------ response ---------------
type MeasureResponse struct {
	Status int64 `json:"status"`
	Body struct {
		MeasureGroup []struct {
			Date    int64 `json:"date"`
			Created int64 `json:"created"`
			Measure []struct {
				Value float64 `json:"value"`
				Unit  int32   `json:"unit"`
			} `json:"measures"`
		} `json:"measuregrps"`
	} `json:"body"`
}

type TokenResponse struct {
	Error []struct {
		message string
	}
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
type Slack struct {
	Text     string `json:"text"`
	Username string `json:"username"`
}


// ------------ endpoints ---------------

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



func auth(c echo.Context) error {
	u := *authUrl
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientId)
	q.Set("state", "diet")
	q.Set("scope", "user.metrics")
	redirectUrl := *selfUrl
	redirectUrl.Path = path.Join(redirectUrl.Path, "user/add")
	q.Set("redirect_uri", redirectUrl.String())
	u.RawQuery = q.Encode()

	_, err := http.Get(u.String())
	if err != nil {
		return c.String(http.StatusUnauthorized, "OK")
	}

	return c.HTML(http.StatusOK,
		fmt.Sprintf("<a href='%s'>Authentication</a>", u.String()))
}

func receiveAuthenticationCode(c echo.Context) error {
	code := c.QueryParam("code")
	log.Infof(code)

	u := *selfUrl
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

	tokenResponse := getAccessToken(code)
	user := User{
		Name:         name,
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
	}
	addUser(user)
	return c.String(http.StatusOK, "OK")
}

func getRanking(c echo.Context) error {
	users := readUser()

	ranking := []UserStatus{}
	for _, user := range (users) {
		ranking = append(ranking, getRatio(user))
	}
	// Add test user
	ranking = append(ranking, UserStatus{
		Ratio: 0.90,
		Name:  "yaseta hito",
	})
	ranking = append(ranking, UserStatus{
		Ratio: 1.1,
		Name:  "futotta hito",
	})

	sort.Slice(ranking, func(i, j int) bool { return ranking[i].Ratio < ranking[j].Ratio })

	ranks := []string{}
	for i, status := range (ranking) {
		s := fmt.Sprintf("%d: %s %.2f", i+1, status.Name, status.Ratio*100)
		ranks = append(ranks, s)
	}

	message := strings.Join(ranks, "\n")
	echoToSlack(message)

	return c.String(http.StatusOK, "OK")
}

// util
func echoToSlack(message string) {
	params, _ := json.Marshal(Slack{
		message,
		name,
	})
	resp, _ := http.PostForm(
		slackWebhookUrl,
		url.Values{"payload": {string(params)}},
	)

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	fmt.Println(string(body))

}

func getRatio(user User) UserStatus {
	t, _ := time.Parse("2006-01-02", "2019-07-01")

	values := url.Values{}
	values.Add("action", "getmeas")
	values.Add("meastype", "1")
	values.Add("category", "1")
	values.Add("access_token", user.AccessToken)
	values.Add("startdate", string(t.Unix()))

	resp, err := http.PostForm(measureUrl.String(), values)
	defer resp.Body.Close()
	if err != nil {
		log.Infof("Failed: %s", user.Name)
		panic(err)
	}

	log.Infof("StatusCode %d", resp.StatusCode)

	if resp.StatusCode >= 400 {
		panic(err)

	}

	body, err := ioutil.ReadAll(resp.Body)
	log.Infof(string(body))

	measure := MeasureResponse{}
	_ = json.Unmarshal(body, &measure)

	if measure.Status == 401 {
		user = refreshToken(user)
	}

	log.Infof("use access %s refresh %s", user.AccessToken, user.RefreshToken)
	var startDate int64 = 0
	var startWeight float64 = 0

	var endDate int64 = 0
	var endWeight float64 = 0

	for _, group := range (measure.Body.MeasureGroup) {
		weight := group.Measure[0].Value / math.Pow(10, float64(-group.Measure[0].Unit))
		log.Infof("name: %s date: %s weight: %f",
			user.Name, time.Unix(group.Created, 0), weight)

		if endDate == 0 {
			endDate = group.Created
			endWeight = weight
		}

		if group.Created < t.Unix() {
			break
		}
		startWeight = weight
		startDate = group.Created
	}

	log.Infof("Start Date: %s, Weight %.4f", time.Unix(startDate, 0), startWeight)
	log.Infof("End Date: %s, Weight %.4f", time.Unix(endDate, 0), endWeight)

	return UserStatus{
		Name:  user.Name,
		Ratio: endWeight / startWeight,
	}

}

func getAccessToken(code string) TokenResponse {
	values := url.Values{}

	redirectUrl := *selfUrl
	redirectUrl.Path = path.Join(redirectUrl.Path, "user/add")

	values.Add("grant_type", "authorization_code")
	values.Add("client_id", clientId)
	values.Add("client_secret", secret)
	values.Add("code", code)
	values.Add("redirect_uri", redirectUrl.String())

	resp, err := http.PostForm(tokenUrl.String(), values)
	defer resp.Body.Close()

	if err != nil {
		log.Error("Failed to get access token")
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	tokenResponse := TokenResponse{}
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		log.Error("Failed to parse token response")
		panic(err)
	}

	return tokenResponse
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

func refreshToken(user User) User {
	values := url.Values{}

	values.Add("grant_type", "refresh_token")
	values.Add("client_id", clientId)
	values.Add("client_secret", secret)
	values.Add("refresh_token", user.RefreshToken)

	resp, err := http.PostForm(tokenUrl.String(), values)
	if err != nil {
		log.Info("Failed to get refresh token")
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Info("Failed to read respong")
		panic(err)
	}
	log.Info("refresh token")
	log.Info(body)

	tokenResponse := TokenResponse{}
	err = json.Unmarshal(body, &tokenResponse)
	log.Infof("access %s refresh %s", tokenResponse.AccessToken, user.RefreshToken)

	user.AccessToken = tokenResponse.AccessToken
	updateUser(user, tokenResponse)

	return user

}

func updateUser(user User, token TokenResponse) {
	users := readUser()
	for _, u := range (users) {
		if user.Name == u.Name {
			u.AccessToken = token.AccessToken
			u.RefreshToken = token.RefreshToken
		}
	}

	userJson, err := json.MarshalIndent(users, "", "    ")
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile(dataFile, userJson, os.ModePerm)
}

func init() {
	selfUrl, _ = url.Parse(os.Getenv("SELF_URL"))
	authUrl, _ = url.Parse("https://account.withings.com/oauth2_user/authorize2")
	tokenUrl, _ = url.Parse("https://account.withings.com/oauth2/token")
	measureUrl, _ = url.Parse("https://wbsapi.withings.net/measure")
}

func main() {
	e := echo.New()
	e.GET("/", home)
	e.GET("/auth", auth)
	e.GET("/ok", finshAuth)
	e.GET("/members", members)
	e.GET("/user/add", receiveAuthenticationCode)
	e.POST("/user/add", saveAccessToken)
	e.GET("/echo", getRanking)

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
