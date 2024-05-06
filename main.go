package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goombaio/namegenerator"
	"github.com/ox-y/GoGmailnator"
	"github.com/sethvargo/go-password/password"
)

func main() {
	fmt.Println("twitch-accounts by xBadApple -  https://github.com/xBadApple")

	if config.CapSolverKey == "your_captcha_key" {
		log.Fatal("It looks like your captcha solver API token isn't configured yet. Change it in the config.go file and run again.")
	}

	createNewAccount()

}

func createNewAccount() {
	randomUsername := getRandomUsername() + "_" + generateRandomID(3)

	trashMailSession := getTrashMailSession()
	randomEmail := trashMailSession.Email

	registerPostData := generateRandomRegisterData(randomUsername, randomEmail)

	fmt.Println("Getting twitch cookies.")
	cookies := getTwitchCookies()

	fmt.Println("Getting kasada code")
	taskResponse := kasadaResolver()

	fmt.Println("Getting local integrity token") // Add proxy later into integrity
	getIntegrityOption(taskResponse)

	integrityData := integrityGetToken(taskResponse, cookies)
	if integrityData.Token == "" {
		log.Fatal("Unable to get register token!")
	}

	fmt.Println("Creating account...")
	registerPostData.IntegrityToken = integrityData.Token
	registerData, err := registerFinal(cookies, registerPostData, taskResponse.Solution["user-agent"])
	if err == nil {
		log.Fatal(err)
	}

	userId := registerData.UserId
	accessToken := registerData.AccessToken

	fmt.Println("Account created!")
	fmt.Println("UserID:", userId, "AccessToken:", accessToken)

	fmt.Println("Waiting email verification ...")
	time.Sleep(time.Second * 2) // Sleep for 2 seconds because twitch verification email can have some delay
	verifyCode, _ := getVerificationCode(trashMailSession)

	fmt.Printf("%+v", verifyCode)
}

func getRandomUsername() string {
	nameGenerator := namegenerator.NewNameGenerator(time.Now().UTC().UnixNano())

	name := strings.Replace(nameGenerator.Generate(), "-", "", -1)
	return name
}

func generateRandomID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	bytes := make([]byte, length)
	for i := range bytes {
		index := rand.Intn(len(charset))
		bytes[i] = charset[index]
	}
	return string(bytes)
}

func getEmail(username string) string {
	return fmt.Sprintf("%s@%s", username, config.EmailDomain) // Unused right now
}

func generateRandomRegisterData(uname string, email string) RandomRegisterData {
	return RandomRegisterData{
		Username:       uname,
		Password:       getRandomPassword(),
		Birthday:       generateRandomBirthday(),
		Email:          email,
		ClientID:       config.TwitchClientID,
		IntegrityToken: "",
	}
}

func getRandomPassword() string {
	res, err := password.Generate(32, 1, 1, false, false)
	if err != nil {
		log.Fatal(err)
	}

	return res
}

func generateRandomBirthday() Birthday {
	return Birthday{
		Day:   rand.Intn(30) + 1,
		Month: rand.Intn(12) + 1,
		Year:  rand.Intn(30) + 1970,
	}
}

func getTwitchCookies() map[string]string {
	cookiesMap := make(map[string]string)
	httpClient := &http.Client{}
	var proxyURL *url.URL

	if config.Proxy == "your_proxy" {
		fmt.Println("!! There is no proxy configuration found. The requests are going to be handled without any proxy !!")
	} else {
		var err error
		proxyURL, err = url.Parse(config.Proxy)
		if err != nil {
			log.Fatal("Error parsing proxy URL:", err)
		}

		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	req, err := http.NewRequest("GET", "https://twitch.tv", nil)
	if err != nil {
		log.Fatal("Error creating the request:", err)
	}

	req.Header.Set("User-Agent", "current_useragent")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatal("Error making request:", err)
	}
	defer resp.Body.Close()

	for _, cookieData := range resp.Header["Set-Cookie"] {
		cookie := strings.Split(cookieData, ";")[0]
		cookiesMap[strings.Split(cookie, "=")[0]] = strings.Split(cookie, "=")[1]
	}

	return cookiesMap
}

func kasadaResolver() ResultTaskResponse {
	taskResponse := createKasadaTask()
	time.Sleep(time.Second * 1)
	taskResult := getTaskResult(taskResponse.TaskId)

	fmt.Println(taskResult)

	return taskResult
}

func createKasadaTask() CreateTaskResponse {
	requestBody := CreateKasadaTask{
		ApiKey: config.CapSolverKey,
		Task: Task{
			Type:   "KasadaCaptchaSolver",
			Pjs:    "https://k.twitchcdn.net/149e9513-01fa-4fb0-aad4-566afd725d1b/2d206a39-8ed7-437e-a3be-862e0f06eea3/p.js",
			CdOnly: false,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post("https://salamoonder.com/api/createTask", "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(body))

	taskResp := CreateTaskResponse{}
	json.Unmarshal(body, &taskResp)

	return taskResp
}

func getTaskResult(taskId string) ResultTaskResponse {
	task := GetTaskResult{TaskId: taskId}

	jsonBody, _ := json.Marshal(task)

	resp, _ := http.Post("https://salamoonder.com/api/getTaskResult", "application/json", bytes.NewBuffer(jsonBody))
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	taskResponse := ResultTaskResponse{}

	json.Unmarshal(body, &taskResponse)

	return taskResponse
}

func getIntegrityOption(taskResponse ResultTaskResponse) {
	client := &http.Client{}

	req, err := http.NewRequest("OPTIONS", "https://passport.twitch.tv/integrity", nil)
	if err != nil {
		log.Fatal("Error creating request:", err)
	}

	req.Header.Set("User-Agent", taskResponse.Solution["user-agent"])
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "x-kpsdk-cd,x-kpsdk-ct")
	req.Header.Set("Referer", "https://www.twitch.tv/")
	req.Header.Set("Origin", "https://www.twitch.tv")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error sending request:", err)
	}

	defer resp.Body.Close()

	// Print the response status code
	fmt.Println("Response Status:", resp.Status)
}

func integrityGetToken(taskResponse ResultTaskResponse, cookies map[string]string) Token {
	client := &http.Client{}

	req, _ := http.NewRequest("POST", "https://passport.twitch.tv/integrity", nil)

	req.Header.Set("User-Agent", taskResponse.Solution["user-agent"])
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://www.twitch.tv/")
	req.Header.Set("x-kpsdk-ct", taskResponse.Solution["x-kpsdk-ct"])
	req.Header.Set("x-kpsdk-cd", taskResponse.Solution["x-kpsdk-cd"])
	req.Header.Set("Origin", "https://www.twitch.tv")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Content-Length", "0")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error making request:", err)
	}
	defer resp.Body.Close()

	for _, cookieData := range resp.Header["Set-Cookie"] {
		cookie := strings.Split(cookieData, ";")[0]
		cookies[strings.Split(cookie, "=")[0]] = strings.Split(cookie, "=")[1]
	}

	body, _ := io.ReadAll(resp.Body)

	token := Token{}
	json.Unmarshal(body, &token)

	return token
}

func registerFinal(cookies map[string]string, postParams RandomRegisterData, userAgent string) (*AccountRegisterResponse, error) {
	var cookiesString string
	for key, value := range cookies {
		cookiesString += key + "=" + value + "; "
	}

	client := &http.Client{}

	jsonBody, _ := json.Marshal(postParams)

	req, _ := http.NewRequest("POST", "https://passport.twitch.tv/protected_register", bytes.NewBuffer(jsonBody))

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://www.twitch.tv/")
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Origin", "https://www.twitch.tv")
	req.Header.Set("DNT", "1")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", cookiesString)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")

	resp, _ := client.Do(req)

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 {
		registerResponse := &AccountRegisterResponse{}
		json.Unmarshal(body, registerResponse)

		return registerResponse, nil
	} else {
		return nil, errors.New(string(body))
	}

}

func getTrashMailSession() *MailnatorData {
	var sess GoGmailnator.Session

	// session will expire after a few hours
	err := sess.Init(nil)
	if err != nil {
		panic(err)
	}

	// calling sess.GenerateEmailAddress or sess.RetrieveMail with a dead session will cause an error
	isAlive, err := sess.IsAlive()
	if err != nil {
		panic(err)
	}

	if isAlive {
		fmt.Println("Session is alive.")
	} else {
		fmt.Println("Session is dead.")
		return nil
	}

	emailAddress, err := sess.GenerateEmailAddress()
	if err != nil {
		panic(err)
	}

	fmt.Println("Email address is " + emailAddress + ".")

	mailData := &MailnatorData{
		Session: sess,
		Email:   emailAddress,
	}

	return mailData

	/*
		emails, err := sess.RetrieveMail(emailAddress)
		if err != nil {
			panic(err)
		}

		for _, email := range emails {
			fmt.Printf("From: %s, Subject: %s, Time: %s\n", email.From, email.Subject, email.Time)
		}
	*/
}

func getVerificationCode(mailData *MailnatorData) (string, error) {
	emails, err := mailData.Session.RetrieveMail(mailData.Email)
	if err != nil {
		panic(err)
	}

	var verificationCode string
	for _, email := range emails {
		if strings.Contains(email.Subject, "Twitch") {
			split := strings.Split(email.Subject, "–")[0]
			verificationCode = strings.TrimSpace(split)
			break
		}
	}

	if verificationCode == "" {
		return "", errors.New("there is no twitch email")
	}

	fmt.Println("Verification code:", verificationCode)

	return verificationCode, nil
}
