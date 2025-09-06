package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const HA_URL = "http://homeassistant.local:8123"

func main() {
	var username string
	var password string
	var haToken string
	flag.StringVar(&username, "u", "", "-u username")
	flag.StringVar(&password, "p", "", "-p password")
	flag.StringVar(&haToken, "t", "", "-t long live token")
	flag.Parse()

	if username == "" || password == "" || haToken == "" {
		log.Fatal("missing USERNAME, PASSWORD, HA_TOKEN env")
	}

	// 启动时先跑一次（可删）
	task(username, password, haToken)

	// ticker := time.NewTicker(15 * time.Minute)
	// defer ticker.Stop()

	// // 循环执行
	// for range ticker.C {
	// 	task(username, password, haToken)
	// }
}

func task(username string, password string, haToken string) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Step 1: get login page, extract token
	log.Println("Tring to get form token......")
	loginPage, err := client.Get("https://myenecle.com/Login")
	if err != nil {
		log.Fatal("failed to fetch login page:", err)
	}
	body, _ := io.ReadAll(loginPage.Body)
	defer loginPage.Body.Close()

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
	// usage
	usage := extractUsage(string(mpbody))
	cost := extractCost(string(mpbody))
	annualUsageMap := extractAnnualUsageMap(string(mpbody))
	extractAnnualUsage := extractAnnualUsage(string(mpbody))

	log.Println("Gas usage:", usage)
	log.Println("Gas cost:", cost)
	log.Println("Annual usage map:", string(annualUsageMap))
	log.Println("Annual usage:", extractAnnualUsage)

	if err := pushAllEnergySensors(client, haToken, usage, cost, extractAnnualUsage); err != nil {
		log.Fatal(err)
	}
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
func extractUsage(htmlBody string) float64 {
	// 正则：前面必须有指定的 span，捕获 <em> 内的数字
	re := regexp.MustCompile(`<span>&#x3054;&#x4F7F;&#x7528;&#x91CF;</span>\s*<span><em>([\d.]+)</em>`)
	m := re.FindStringSubmatch(htmlBody)
	if len(m) > 1 {
		// 转换为 float64
		f, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			fmt.Println("Convert err", err)
			return 0
		}

		fmt.Println("float64:", f)
		return f
	}
	return 0
}

// // extractUsage 提取 HTML 中 <h3> 标签内的数字
// // extractCost 提取 HTML 中 <h3> 标签内的数字
// func extractCost(htmlBody string) float64 {
// 	re := regexp.MustCompile(`<h3 class="idxprc__sum">([\d,]+)円</h3>`)
// 	m := re.FindStringSubmatch(htmlBody)
// 	if len(m) > 1 {
// 		// 去掉千位分隔符
// 		numStr := strings.ReplaceAll(m[1], ",", "")
// 		f, err := strconv.ParseFloat(numStr, 64)
// 		if err != nil {
// 			fmt.Println("Convert err:", err)
// 			return 0
// 		}

// 		fmt.Println("float64:", f)
// 		return f
// 	}
// 	return 0
// }

// extractCost 提取 HTML 中 <h3> 标签内的数字
func extractCost(htmlBody string) float64 {
	re := regexp.MustCompile(`<h3 class="idxprc__sum">([\d,]+)円</h3>`)
	m := re.FindStringSubmatch(htmlBody)
	if len(m) > 1 {
		// 去掉千位分隔符
		numStr := strings.ReplaceAll(m[1], ",", "")
		f, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			fmt.Println("Convert err:", err)
			return 0
		}
		return f
	}
	return 0
}

// parse data from html
func getAnnualUsagesData(htmlBody string) map[string]interface{} {
	re := regexp.MustCompile(`data:\s*(\{[\s\S]*?\})\s*,\s*options:`)
	match := re.FindStringSubmatch(htmlBody)
	var result map[string]interface{}
	if len(match) == 0 {
		return result
	}
	jsObject := match[1]
	// fmt.Println(jsObject)

	jsObject = strings.ReplaceAll(jsObject, "'", `"`) // 单引号 -> 双引号

	// 正则：给 key 加引号
	keyRe := regexp.MustCompile(`(\w+):`)
	jsObject = keyRe.ReplaceAllString(jsObject, `"$1":`)

	// 去掉可能的末尾逗号
	jsObject = strings.ReplaceAll(jsObject, ",}", "}")
	jsObject = strings.ReplaceAll(jsObject, ",]", "]")

	err := json.Unmarshal([]byte(jsObject), &result)
	if err != nil {
		log.Fatal("JSON parse error:", err)
	}
	return result
}

// 获取年度汇报
func extractAnnualUsageMap(htmlBody string) []byte {
	result := getAnnualUsagesData(htmlBody)
	newMap := make(map[string]interface{})
	// 提取 labels
	newMap["month"] = result["labels"]
	// 提取 datasets[0].data
	datasets := result["datasets"].([]interface{})
	if len(datasets) > 0 {
		first := datasets[0].(map[string]interface{})
		newMap["datasets"] = first["data"]
	}
	data, _ := json.Marshal(newMap)
	return data
}

// 获取年度汇报总结
func extractAnnualUsage(htmlBody string) float64 {
	result := getAnnualUsagesData(htmlBody) // 返回 map[string]interface{}
	var total float64
	// 提取 datasets
	datasets, ok := result["datasets"].([]interface{})
	if !ok || len(datasets) == 0 {
		return 0
	}
	first, ok := datasets[0].(map[string]interface{})
	if !ok {
		return 0
	}
	dataList, ok := first["data"].([]interface{})
	if !ok {
		return 0
	}
	// 累加每个月的数据
	for _, v := range dataList {
		switch val := v.(type) {
		case float64:
			total += val
		case string:
			// 字符串转 float64
			f, err := strconv.ParseFloat(strings.ReplaceAll(val, ",", ""), 64)
			if err == nil {
				total += f
			}
		}
	}
	return total
}

// pushEnergySensor 推送一个能源面板可识别的传感器
func pushEnergySensor(client *http.Client, haToken, entity string, state float64, unit, deviceClass string) error {
	data := map[string]interface{}{
		"state": state,
		"attributes": map[string]interface{}{
			"unit_of_measurement": unit,
			"device_class":        deviceClass,
			"state_class":         "total_increasing", // 必须是 total_increasing 才能被 Energy Panel 识别
			"friendly_name":       entity,             // 用 entity 名称作为 friendly_name
		},
	}

	payload, _ := json.Marshal(data)
	url := fmt.Sprintf("%s/api/states/%s", HA_URL, entity)

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Authorization", "Bearer "+haToken)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to push to HA: %w", err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	log.Println("HA Response:", res.Status, string(body))

	if res.StatusCode != 200 {
		return fmt.Errorf("HA API returned %s", res.Status)
	}

	return nil
}

// pushAllEnergySensors 推送燃气用量、费用、年度累计三个传感器
func pushAllEnergySensors(client *http.Client, haToken string, usage, cost, annualUsage float64) error {
	// 燃气用量
	if err := pushEnergySensor(client, haToken, "sensor.enecle_usage", usage, "m³", "gas"); err != nil {
		return err
	}

	// 燃气费用
	if err := pushEnergySensor(client, haToken, "sensor.enecle_cost", cost, "JPY", "monetary"); err != nil {
		return err
	}

	// 年度累计燃气量
	if err := pushEnergySensor(client, haToken, "sensor.enecle_annual_usage", annualUsage, "m³", "gas"); err != nil {
		return err
	}

	return nil
}
