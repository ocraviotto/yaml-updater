package cmd

import (
	"crypto/tls"
	"net/http"

	"github.com/ocraviotto/go-scm/scm"
	"github.com/ocraviotto/go-scm/scm/factory"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

func createClientFromViper() (*scm.Client, error) {
	authToken := viper.GetString(authTokenFlag)
	driver := viper.GetString(driverFlag)
	apiEndpoint := viper.GetString(apiEndpointFlag)
	username := viper.GetString(usernameFlag)
	if viper.GetBool(insecureFlag) {
		return factory.NewClient(
			driver,
			apiEndpoint,
			"",
			factory.Client(makeInsecureClient(authToken)))

	}
	return factory.NewClient(
		driver,
		apiEndpoint,
		authToken,
		factory.SetUsername(username))
}

func makeInsecureClient(token string) *http.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return &http.Client{
		Transport: &oauth2.Transport{
			Source: ts,
			Base: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}
