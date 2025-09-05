package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"

	"golang.org/x/net/html"
)

func main() {
	var username string
	var password string
	var haToken string
	flag.StringVar(&username, "u", "", "-u username")
	flag.StringVar(&password, "p", "", "-p password")
	flag.StringVar(&haToken, "t", "", "-t long live token")
	flag.Parse()

	haURL := "http://homeassistant:8123"

	if username == "" || password == "" || haToken == "" {
		log.Fatal("missing USERNAME, PASSWORD, HA_TOKEN env")
	}

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Step 1: get login page, extract token
	log.Println("Tring to get form token......")
	loginPage, err := client.Get("https://myenecle.com/Login")
	if err != nil {
		log.Fatal("failed to fetch login page:", err)
	}
	body, _ := io.ReadAll(loginPage.Body)
	loginPage.Body.Close()

	token := extractToken(string(body))
	log.Println("Fetched token:", token)

	// Step 2: login
	log.Println("Tring to login......")
	form := url.Values{}
	form.Add("__RequestVerificationToken", token)
	form.Add("MailAddress", username)
	form.Add("Password", password)
	encodedForm := form.Encode() // 转成 "key1=value1&key2=value2" 形式
	buffer := bytes.NewBufferString(encodedForm)
	req, err := http.NewRequest("POST", "https://myenecle.com/Login", buffer)
	if err != nil {
		log.Fatal("failed to create request:", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded") // 设置表单类型

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("failed to login:", err)
	}
	defer resp.Body.Close()

	// 正则匹配 <div class="validation-summary-errors"> 里所有 <li> 内容
	re := regexp.MustCompile(`<div class="validation-summary-errors"[^>]*>.*?<ul>.*?<li>(.*?)</li>`)
	body, _ = io.ReadAll(resp.Body)
	matches := re.FindAllStringSubmatch(string(body), -1)
	for _, m := range matches {
		if len(m) > 1 {
			decoded := html.UnescapeString(m[1])
			log.Println("Login fail something wrong with: " + decoded)
			return
		}
	}

	// 现在 cookies 都被 Jar 保存了
	u, _ := url.Parse("https://myenecle.com")
	for _, c := range jar.Cookies(u) {
		log.Printf("{ Key: %s, Value: %s }\n", c.Name, c.Value)
	}

	// Step 3: fetch MyPageTop
	mypage, err := client.Get("https://myenecle.com/MyPage/MyPageTop")
	if err != nil {
		log.Fatal("failed to fetch mypage:", err)
	}
	mpbody, _ := io.ReadAll(mypage.Body)
	defer mypage.Body.Close()

	// log.Println(string(mpbody))

	usage := extractUsage(string(mpbody))
	if usage == "" {
		log.Println("Get invalid gas usage.")
		return
	}
	log.Println("Gas usage:", usage)

	// Step 4: push to Home Assistant
	state := "error"
	attrs := `"error_msg": "login failed"` // 这里不需要 fmt.Sprintf，如果只是固定字符串
	if usage != "" {
		state = usage
		attrs = `"unit_of_measurement":"m3", "friendly_name":"Enecle usage"`
	}

	payload := fmt.Sprintf(`{"state":"%s","attributes":{%s}}`, state, attrs)
	req, _ = http.NewRequest("POST", haURL+"/api/states/sensor.enecle_usage", bytes.NewBuffer([]byte(payload)))
	req.Header.Set("Authorization", "Bearer "+haToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Fatal("failed to push to HA:", err)
	}
	_, _ = io.ReadAll(res.Body)
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

// extractUsage 提取 HTML 中 <em> 标签内的数字
func extractUsage(htmlBody string) string {
	// 正则：前面必须有指定的 span，捕获 <em> 内的数字
	re := regexp.MustCompile(`<span>&#x3054;&#x4F7F;&#x7528;&#x91CF;</span>\s*<span><em>([\d.]+)</em>`)
	match := re.FindStringSubmatch(htmlBody)
	if len(match) > 1 {
		return match[1] // 输出 1.1
	}
	return ""
}
