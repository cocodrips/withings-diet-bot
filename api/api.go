package api

import (
	"github.com/labstack/echo"
	"net/http"
	"github.com/cocodrips/withings-diet-bot/withings"
	"strings"
	"fmt"
	"sort"
	"github.com/cocodrips/withings-diet-bot/slack"
	"os"
)

type UserStatus struct {
	Name  string
	Ratio float64
}

func home(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
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
	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].Ratio < ranking[j].Ratio
	})

	//
	ranks := []string{}
	ranks = append(ranks,
		fmt.Sprintf("Today's ranking (Compare to %s)",
			os.Getenv("START_DATE")))
	for i, status := range (ranking) {
		s := fmt.Sprintf("%d: %s %.2f％", i+1, status.Name, status.Ratio*100)
		ranks = append(ranks, s)
	}

	message := strings.Join(ranks, "\n")
	slack.EchoToSlack(message)

	return c.String(http.StatusOK, "OK")
}

func Rooting(e *echo.Echo) {
	e.GET("/", home)
	e.GET("/auth", auth)
	e.GET("/members", members)
	e.GET("/user/add", receiveAuthenticationCode)
	e.POST("/user/add", saveAccessToken)
	e.GET("/echo", getRanking)
}
