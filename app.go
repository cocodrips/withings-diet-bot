package main

import (
	"github.com/labstack/echo"
	"github.com/labstack/gommon/log"
	"os"
	"fmt"
	"github.com/labstack/echo/middleware"
	"github.com/cocodrips/withings-diet-bot/api"
)

var (
	port     string = os.Getenv("PORT")
	dataFile string = "diet-token.json"
)

func main() {
	fp, err := os.OpenFile("/var/log/app/access.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	dataFile, err := os.OpenFile(dataFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	dataFile.Close()

	e := echo.New()
	api.Rooting(e)

	// Logger
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "method=${method}, uri=${uri}, status=${status}\n",
		Output: fp,
	}))
	log.SetOutput(fp)
	e.Logger.SetLevel(log.INFO)

	e.Use(middleware.Recover())

	e.Logger.Fatal(e.Start(fmt.Sprintf(":%s", port)))
}
