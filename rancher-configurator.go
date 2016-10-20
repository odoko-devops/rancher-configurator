package main

import (
	"net/http"
	"time"
	"bytes"
	"fmt"
	"encoding/json"
	"log"
	"io"
	"os"
)

func main() {
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	email := os.Getenv("EMAIL")
	dockerHost := os.Getenv("DOCKER_HOST")
	rancherHost := os.Getenv("RANCHER_HOST")

	accessKey, secretKey, err := configureRancher(username, password, email, dockerHost, rancherHost)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ACCESS_KEY:%s SECRET_KEY:%s\n", accessKey, secretKey)
}
func configureRancher(username, password, email, dockerHost, rancherHost string) (string, string, error) {
	envId := getEnvironmentId(rancherHost)
	accessKey, secretKey, err := getApiKeys(rancherHost, envId)
	if err != nil {
		return "","", err
	}

	setApiHost(rancherHost)
	registryId, err := registerRegistry(rancherHost, envId, dockerHost)
	if err != nil {
		return "","", err
	}
	err = registryCredentials(rancherHost, envId, registryId, username, password, email)
	if err != nil {
		return "","", err
	}

	err = enableAuth(rancherHost, username, password)
	if err != nil {
		return "","", err
	}
	log.Println("Rancher configured.")
	return accessKey, secretKey, nil
}

func getEnvironmentId(rancherHost string) string {
	log.Println("Getting environment ID...")
	type EnvData struct {
		Data []struct {
			Id string `json:"id"`
			Name string `json:"name"`
		}
	}

	envUrl := fmt.Sprintf("http://%s/v1/accounts", rancherHost)
	log.Printf("Trying Rancher on %s...\n", envUrl)

	for {
		resp, err := http.Get(envUrl)
		if (err != nil) {
			time.Sleep(5*time.Second)
			println("Waiting for Rancher...")
			continue
		}
		envData := EnvData{}
		err = json.NewDecoder(resp.Body).Decode(&envData)

		if (err != nil && err!=io.EOF) {
			time.Sleep(5*time.Second)
			println("Waiting for Rancher...")
			continue
		}
		for _, account := range envData.Data {
			if account.Name == "Default" {
				return account.Id
			}
		}
		time.Sleep(5 * time.Second)
	}
}

func getApiKeys(rancherHost string, envId string) (string, string, error) {

	type KeyData struct {
		PublicValue string
		SecretValue string
	}
	data := map[string]interface{} {
		"type": "apikey",
		"accountId": envId,
		"name": "api_key",
		"description": "api_key",
		"created": nil,
		"kind": nil,
		"removed": nil,
		"uuid": nil,
	}
	log.Println("Getting API keys...")
	byteData, err := json.Marshal(&data)
	if err != nil {
		return "","",err
	}
	apiKeyUrl := fmt.Sprintf("http://%s/v1/projects/%s/apikey", rancherHost, envId)
	resp, err := http.Post(apiKeyUrl, "application/json", bytes.NewBuffer(byteData))
	if err != nil {
		return "","",err
	}
	keyData := KeyData{}
	err = json.NewDecoder(resp.Body).Decode(&keyData)
	if err != nil {
		return "","",err
	}
	return keyData.PublicValue, keyData.SecretValue, nil
}

func enableAuth(rancherHost, username, password string) error {
	data := map[string]interface{}{
		"accessMode":"unrestricted",
		"name": username,
		"id": nil,
		"type":"localAuthConfig",
		"enabled": true,
		"password": password,
		"username": username,
	}

	byteData, _ := json.Marshal(&data)
	url := fmt.Sprintf("http://%s/v1/localauthconfig", rancherHost)
	_, err := http.Post(url, "application/json", bytes.NewBuffer(byteData))
	if err != nil {
		return err
	}

	log.Println("Rancher auth enabled")
	return nil
}

func registerRegistry(rancherHost, envId, dockerHost string) (string, error) {

	data := map[string]interface{}{
		"type": "registry",
		"serverAddress": dockerHost,
		"blockDevicePath": "",
		"created": nil,
		"description": "Private Docker Registry",
		"driverName": nil,
		"externalId": nil,
		"kind": nil,
		"name": nil,
		"removed": nil,
		"uuid": nil,
		"volumeAccessMode": nil,
	}

	byteData, _ := json.Marshal(&data)
	type RegistryData struct{ Id string `json:"id"` }

	registryUrl := fmt.Sprintf("http://%s/v1/projects/%s/registry", rancherHost, envId)
	resp, err := http.Post(registryUrl, "application/json", bytes.NewBuffer(byteData))
	if err != nil {
		return "", err
	}
	registryData := RegistryData{}
	err = json.NewDecoder(resp.Body).Decode(&registryData)
	if err!= nil {
		return "", err
	}
	println("Docker Registry registered")
	return registryData.Id, nil
}

func registryCredentials(rancherHost, envId, registryId, username, password, email string) error {

	data := map[string]interface{}{
		"type": "registryCredential",
		"registryId": registryId,
		"email": email,
		"publicValue": username,
		"secretValue": password,
		"created": nil,
		"description": nil,
		"kind": nil,
		"name": nil,
		"removed": nil,
		"uuid": nil,
	}
	credentialsUrl := fmt.Sprintf("http://%s/v1/projects/%s/registrycredential", rancherHost, envId)
	byteData, _ := json.Marshal(&data)

	_, err := http.Post(credentialsUrl, "application/json", bytes.NewBuffer(byteData))
	if err != nil {
		return err
	}

	log.Println("Docker Registry credentials configured")
	return nil
}

func setApiHost(rancherHost string) error {
	type ApiData struct {
		Id    string `json:"id"`
		Links struct {
			      Self string `json:"self"`
		      } `json:"links"`
	}
	url := fmt.Sprintf("http://%s/v1/settings/api.host", rancherHost)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	apiData := ApiData{}
	err = json.NewDecoder(resp.Body).Decode(&apiData)
	if err != nil {
		return err
	}
	apiUrl := apiData.Links.Self
	apiId := apiData.Id

	data := map[string]interface{}{
		"id": apiId,
		"type": "activeSetting",
		"name": "api.host",
		"activeValue": nil,
		"inDb": false,
		"source": nil,
		"value": fmt.Sprintf("http://%s", rancherHost),
	}
	byteData, _ := json.Marshal(&data)
	req, err := http.NewRequest("PUT", apiUrl, bytes.NewBuffer(byteData))
	if err != nil {
		return err
	}
	client := &http.Client{}
	client.Do(req)

	log.Print("API Host set.")
	return nil
}
