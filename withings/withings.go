package withings

import (
	"net/url"
	"os"
	"path"
	"github.com/labstack/gommon/log"
	"fmt"
	"net/http"
	"io/ioutil"
	"encoding/json"
	"time"
	"math"
)

var (
	clientId   string = os.Getenv("CLIENT_ID")
	secret     string = os.Getenv("SECRET")
	startDate  string = os.Getenv("START_DATE")
	dataFile   string = "diet-token.json"
	selfUrl    *url.URL
	authUrl    *url.URL
	tokenUrl   *url.URL
	measureUrl *url.URL
)

type User struct {
	Name         string `json:"name"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type TokenResponse struct {
	Error []struct {
		message string
	}
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

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

func init() {
	selfUrl, _ = url.Parse(os.Getenv("SELF_URL"))
	authUrl, _ = url.Parse("https://account.withings.com/oauth2_user/authorize2")
	tokenUrl, _ = url.Parse("https://account.withings.com/oauth2/token")
	measureUrl, _ = url.Parse("https://wbsapi.withings.net/measure")
}

func GetAuthUrl() string {
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

	return u.String()
}

func GetAccessTokenForm(code string) string {
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

	return html
}

func GetAccessToken(code string) TokenResponse {
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

func RefreshToken(user User) User {
	log.Info("---Refresh token---")
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
		log.Info("Failed to read respose")
		panic(err)
	}

	tokenResponse := TokenResponse{}
	err = json.Unmarshal(body, &tokenResponse)

	log.Infof("%s access %s refresh %s",
		user.Name,
		tokenResponse.AccessToken,
		tokenResponse.RefreshToken)

	user = UpdateUser(user, tokenResponse)
	return user
}

func GetMeasure(user User) MeasureResponse {
	t, _ := time.Parse("2006-01-02", startDate)

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

	if resp.StatusCode >= 400 {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)

	measure := MeasureResponse{}
	_ = json.Unmarshal(body, &measure)

	return measure
}

func GetRatio(user User) (string, float64) {
	// Start date
	t, _ := time.Parse("2006-01-02", startDate)
	measure := GetMeasure(user)
	if measure.Status == 401 {
		user = RefreshToken(user)
		measure = GetMeasure(user)
	}

	var startDate int64 = 0
	var startWeight float64 = 0

	var endDate int64 = 0
	var endWeight float64 = 0
	for _, group := range (measure.Body.MeasureGroup) {
		weight := group.Measure[0].Value / math.Pow(10, float64(-group.Measure[0].Unit))

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

	log.Infof("Start Date: %s", time.Unix(startDate, 0))
	log.Infof("End Date: %s", time.Unix(endDate, 0))

	return user.Name, endWeight / startWeight
}
