package withings

import (
	"io/ioutil"
	"github.com/labstack/gommon/log"
	"encoding/json"
	"os"
)

func ReadUser() []User {
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

func AddUser(user User) {
	users := ReadUser()
	users = append(users, user)

	userJson, err := json.MarshalIndent(users, "", "    ")
	if err != nil {
		panic(err)
	}

	log.Infof("Registed %s: access %s refresh %s",
		user.Name,
		user.AccessToken,
		user.RefreshToken)

	ioutil.WriteFile(dataFile, userJson, os.ModePerm)
}

func SaveAccessToken(code string, name string) {
	tokenResponse := GetAccessToken(code)
	user := User{
		Name:         name,
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
	}
	AddUser(user)
}

func UpdateUser(user User, token TokenResponse) User {
	users := ReadUser()
	for i, u := range (users) {
		if user.Name == u.Name {
			users[i].AccessToken = token.AccessToken
			users[i].RefreshToken = token.RefreshToken
		}
	}

	userJson, err := json.MarshalIndent(users, "", "    ")
	if err != nil {
		panic(err)
	}

	ioutil.WriteFile(dataFile, userJson, os.ModePerm)

	user.AccessToken = token.AccessToken
	user.RefreshToken = token.RefreshToken
	return user
}
