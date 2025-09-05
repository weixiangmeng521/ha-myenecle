package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	username := os.Getenv("USERNAME")
	password := os.Getenv("PASSWORD")
	haToken := os.Getenv("HA_TOKEN")
	haURL := "http://homeassistant:8123"

	if username == "" || password == "" || haToken == "" {
		log.Fatal("missing USERNAME, PASSWORD, HA_TOKEN env")
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Step 1: get login page, extract token
	loginPage, err := client.Get("https://myenecle.com/Login")
	if err != nil {
		log.Fatal("failed to fetch login page:", err)
	}
	body, _ := ioutil.ReadAll(loginPage.Body)
	loginPage.Body.Close()

	token := extractToken(string(body))
	log.Println("Fetched token:", token)

	// Step 2: login
	form := url.Values{}
	form.Add("__RequestVerificationToken", token)
	form.Add("MailAddress", username)
	form.Add("Password", password)

	resp, err := client.PostForm("https://myenecle.com/Login", form)
	if err != nil {
		log.Fatal("failed to login:", err)
	}
	_, _ = ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	// Step 3: fetch MyPageTop
	mypage, err := client.Get("https://myenecle.com/MyPage/MyPageTop")
	if err != nil {
		log.Fatal("failed to fetch mypage:", err)
	}
	mpbody, _ := ioutil.ReadAll(mypage.Body)
	mypage.Body.Close()

	usage := extractUsage(string(mpbody))
	log.Println("Gas usage:", usage)

	// Step 4: push to Home Assistant
	state := "error"
	attrs := fmt.Sprintf(`"error_msg": "login failed"`)
	if usage != "" {
		state = usage
		attrs = `"unit_of_measurement":"m3", "friendly_name":"Enecle usage"`
	}

	payload := fmt.Sprintf(`{"state":"%s","attributes":{%s}}`, state, attrs)
	req, _ := http.NewRequest("POST", haURL+"/api/states/sensor.enecle_usage", bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Authorization", "Bearer "+haToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("failed to push to HA:", err)
	}
	_, _ = ioutil.ReadAll(res.Body)
	res.Body.Close()
	log.Println("Updated Home Assistant sensor.")
}

// 提取 __RequestVerificationToken
func extractToken(htmlBody string) string {
	re := regexp.MustCompile(`name="__RequestVerificationToken"[^>]*value="([^"]+)"`)
	m := re.FindStringSubmatch(htmlBody)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// 提取 “ご使用量 ... m3”
func extractUsage(htmlBody string) string {
	// decode HTML entities: we only care about numbers before "m3"
	doc, err := html.Parse(strings.NewReader(htmlBody))
	if err != nil {
		return ""
	}
	var f func(*html.Node) string
	f = func(n *html.Node) string {
		if n.Type == html.TextNode && strings.Contains(n.Data, "m3") {
			return strings.TrimSpace(strings.ReplaceAll(n.Data, "m3", ""))
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if res := f(c); res != "" {
				return res
			}
		}
		return ""
	}
	return f(doc)
}
