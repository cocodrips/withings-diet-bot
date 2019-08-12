package main

import (
	"net/http"
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"os"
	"fmt"
	"github.com/labstack/echo/middleware"
	"strings"
	"sort"

	"github.com/cocodrips/withings-diet-bot/withings"
	"github.com/cocodrips/withings-diet-bot/slack"
)

var (
	botName  string = os.Getenv("BOT_NAME")
	dataFile string = "diet-token.json"
)

type UserStatus struct {
	Name  string
	Ratio float64
}

func home(c echo.Context) error {
	return c.String(http.StatusOK, botName)
}

func members(c echo.Context) error {
	users := withings.ReadUser()

	names := []string{}
	for _, user := range users {
		names = append(names, user.Name)
	}
	return c.String(http.StatusOK, strings.Join(names, ","))
}

func auth(c echo.Context) error {
	u := withings.GetAuthUrl()
	return c.HTML(http.StatusOK,
		fmt.Sprintf("<a href='%s'>Authentication</a>", u))
}

func receiveAuthenticationCode(c echo.Context) error {
	code := c.QueryParam("code")
	html := withings.GetAccessTokenForm(code)
	return c.HTML(http.StatusOK, html)
}

func saveAccessToken(c echo.Context) error {
	code := c.QueryParam("code")
	name := c.FormValue("diet_name")

	withings.SaveAccessToken(code, name)
	return c.String(http.StatusOK, "OK")
}

func getRanking(c echo.Context) error {
	users := withings.ReadUser()

	// Create user
	ranking := []UserStatus{}
	for _, user := range (users) {
		name, ratio := withings.GetRatio(user)
		ranking = append(ranking, UserStatus{
			Name:  name,
			Ratio: ratio,
		})
	}
	// Add test user
	//ranking = append(ranking, UserStatus{
	//	Ratio: 0.90,
	//	Name:  "yaseta hito",
	//})
	//ranking = append(ranking, UserStatus{
	//	Ratio: 1.1,
	//	Name:  "futotta hito",
	//})

	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].Ratio < ranking[j].Ratio
	})

	//
	ranks := []string{}
	for i, status := range (ranking) {
		s := fmt.Sprintf("%d: %s %.2f", i+1, status.Name, status.Ratio*100)
		ranks = append(ranks, s)
	}

	message := strings.Join(ranks, "\n")
	slack.EchoToSlack(message)

	return c.String(http.StatusOK, "OK")
}

func main() {
	e := echo.New()
	e.GET("/", home)
	e.GET("/auth", auth)
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
