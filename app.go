package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"golang.org/x/net/html"
)

const (
	Broker   = "mqtt://core-mosquitto:1883"
	ClientID = "myenecle-clinet"
)

var (
	MOS_USERNAME = "addons"
	MOS_PASSWORD = "aiteab5elia9hee9ahp5chaoG1aegohcahzie9iigaewiaPeiquu1lau9Ho5Ooje"
)

var mqttClient mqtt.Client

func main() {
	var username string
	var password string
	flag.StringVar(&username, "u", "", "-u username")
	flag.StringVar(&password, "p", "", "-p password")
	flag.Parse()

	if username == "" || password == "" || MOS_USERNAME == "" || MOS_PASSWORD == "" {
		log.Fatal("missing USERNAME, PASSWORD, MOS_PASSWORD, MOS_PASSWORD")
	}

	if runtime.GOOS == "darwin" {
		log.Println("Test env.")
		mqttClient = newMQTTClient()
		defer mqttClient.Disconnect(250)
	}

	// 启动时先跑一次（可删）
	task(username, password)

	// ticker := time.NewTicker(15 * time.Minute)
	// defer ticker.Stop()

	// // 循环执行
	// for range ticker.C {
	// 	task(username, password, haToken)
	// }
}

func task(username string, password string) {
	jar, _ := cookiejar.New(nil)
	httpClinet := &http.Client{Jar: jar}

	// Step 1: get login page, extract token
	log.Println("Tring to get form token......")
	loginPage, err := httpClinet.Get("https://myenecle.com/Login")
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

	resp, err := httpClinet.Do(req)
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
	mypage, err := httpClinet.Get("https://myenecle.com/MyPage/MyPageTop")
	if err != nil {
		log.Fatal("failed to fetch mypage:", err)
	}
	mpbody, _ := io.ReadAll(mypage.Body)
	defer mypage.Body.Close()

	//  Step 4: fetch List
	listPage, err := httpClinet.Get("https://myenecle.com/List")
	if err != nil {
		log.Fatal("failed to fetch mypage:", err)
	}
	listBody, _ := io.ReadAll(listPage.Body)
	linkMap, err := extractLinks(string(listBody))
	if err != nil {
		log.Fatal("fail to extra a link tag:", err)
	}
	firstLink := ""
	for _, link := range linkMap {
		if link["text"] == "検針詳細を表示" && firstLink == "" {
			firstLink = link["href"]
			break
		}
	}

	var detailBody string
	if firstLink != "" {
		log.Println("Target link ", firstLink)
		baseLink := "https://myenecle.com" + firstLink
		detailPage, err := httpClinet.Get(baseLink)
		if err != nil {
			log.Fatal("failed to fetch mypage:", err)
		}
		detailBodyBytes, _ := io.ReadAll(detailPage.Body)
		detailBody = string(detailBodyBytes)
	}

	// log.Println(string(mpbody))
	// usage
	request_usage := extractUsage(string(mpbody))
	request_cost := extractCost(string(mpbody))
	last_mon_usage := extractLastMonUsage(detailBody)
	last_mon_cost := extractLastMonCost(detailBody)
	annualUsageMap := extractAnnualUsageMap(string(mpbody))
	usages, total := extractAnnualUsages(string(mpbody))

	log.Println("Request gas usage:", request_usage)
	log.Println("Request gas cost:", request_cost)
	log.Println("Last month gas usage:", last_mon_usage)
	log.Println("Last month gas cost:", last_mon_cost)
	log.Println("Annual usage map:", annualUsageMap)
	log.Println("Annual usage statics:", usages)
	log.Println("Annual usage:", total)

	log.Println("Going to push data at: ", time.Now())
	if err := pushAllEnergySensors(httpClinet, request_usage, request_cost, last_mon_usage, last_mon_cost, total, usages); err != nil {
		log.Println("Push message to sensor err: ", err)
	}

	if runtime.GOOS == "darwin" {
		defer mqttClient.Disconnect(250)
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
			log.Println("Convert err", err)
			return 0
		}

		log.Println("float64:", f)
		return f
	}
	return 0
}

// extractUsage 提取 HTML 中 <em> 标签内的数字
func extractLastMonUsage(htmlBody string) float64 {
	if htmlBody == "" {
		return 0.0
	}
	// 正则：前面必须有指定的 span，捕获 <em> 内的数字
	re := regexp.MustCompile(`(?s)<dt>\s*今回ご使用量\s*</dt>.*?<dd[^>]*>(?:<strong>)?\s*([0-9.]+)\s*m3`)
	match := re.FindStringSubmatch(htmlBody)
	if len(match) > 1 {
		// 转换为 float64
		f, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			log.Println("Convert err", err)
			return 0.0
		}
		log.Println("float64:", f)
		return f
	}
	return 0.0
}

// extractUsage 提取 HTML 中 <em> 标签内的数字
func extractLastMonCost(htmlBody string) float64 {
	if htmlBody == "" {
		return 0.0
	}
	// 正则：前面必须有指定的 span，捕获 <em> 内的数字
	re := regexp.MustCompile(`(?s)<dl[^>]*>\s*<dt>\s*ガス料金合計\s*</dt>.*?<dd[^>]*>\s*([\d,]+)\s*&#x5186;</dd>`)
	m := re.FindStringSubmatch(htmlBody)
	if len(m) > 1 {
		// 去掉千位分隔符
		numStr := strings.ReplaceAll(m[1], ",", "")
		f, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			log.Println("Convert err:", err)
			return 0
		}
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
			log.Println("Convert err:", err)
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
// https://www.mqtt.cn/1101.html
func pushEnergySensor(entity string, state float64, unit, deviceClass string) error {
	if runtime.GOOS == "darwin" {
		log.Println("dont need to push data")
		return nil
	}
	if mqttClient == nil {
		mqttClient = newMQTTClient()
	}

	// 计算 unique_id / device.identifiers
	hash := fmt.Sprintf("%x", md5.Sum([]byte(entity+deviceClass)))
	uniqueID := hash[:8]
	deviceID := hash[:3]

	// 配置 Topic
	// 确保 entity 只包含小写字母、数字和下划线
	safeEntity := strings.ReplaceAll(entity, ".", "_")
	configTopic := fmt.Sprintf("homeassistant/sensor/%s/config", safeEntity)
	stateTopic := fmt.Sprintf("homeassistant/sensor/%s/state", safeEntity)

	// 配置 Payload
	configPayload := fmt.Sprintf(`{
		"device_class": "%s",
		"state_topic": "%s",
		"unit_of_measurement": "%s",
		"value_template": "{{ value_json.state }}",
		"unique_id": "0x%s",
		"device": {
			"identifiers": ["%s"],
			"name": "%s"
		},
		"state_class": "total"
	}`, deviceClass, stateTopic, unit, uniqueID, entity+deviceID, entity)

	// 发布 Discovery 配置
	token := mqttClient.Publish(configTopic, 0, true, configPayload)
	token.WaitTimeout(1 * time.Second)
	if token.Error() != nil {
		return fmt.Errorf("failed to publish config to MQTT: %w", token.Error())
	}

	// 发布状态
	statePayload := fmt.Sprintf(`{"state": %.3f}`, state)
	token = mqttClient.Publish(stateTopic, 0, true, statePayload)
	if token.Error() != nil {
		return fmt.Errorf("failed to publish state to MQTT: %w", token.Error())
	}

	log.Printf("MQTT published entity '%s': state=%.3f %s\n", entity, state, unit)
	return nil
}

// -----------------------------
// 数据结构
// -----------------------------

type MonthlyUsage struct {
	Month string  `json:"month"`
	Value float64 `json:"value"`
}

// extractAnnualUsages 提取每月用量 + 计算总和
func extractAnnualUsages(htmlBody string) ([]MonthlyUsage, float64) {
	result := getAnnualUsagesData(htmlBody) // 你已有的函数，返回 map[string]interface{}
	var total float64
	var usages []MonthlyUsage

	// 提取 labels (月份)
	labels, ok := result["labels"].([]interface{})
	if !ok {
		return usages, 0
	}

	// 提取 datasets
	datasets, ok := result["datasets"].([]interface{})
	if !ok || len(datasets) == 0 {
		return usages, 0
	}
	first, ok := datasets[0].(map[string]interface{})
	if !ok {
		return usages, 0
	}
	dataList, ok := first["data"].([]interface{})
	if !ok {
		return usages, 0
	}

	// 遍历每月数据
	for i, v := range dataList {
		var val float64
		switch t := v.(type) {
		case float64:
			val = t
		case string:
			f, err := strconv.ParseFloat(strings.ReplaceAll(t, ",", ""), 64)
			if err == nil {
				val = f
			}
		}
		if i < len(labels) {
			monthStr, _ := labels[i].(string)
			usages = append(usages, MonthlyUsage{
				Month: monthStr,
				Value: val,
			})
		}
		total += val
	}

	return usages, total

}

// -----------------------------

// 工具函数：月份字符串转数字
// -----------------------------
func monthToNumber(m string) int {
	m = strings.TrimSuffix(m, "月")
	num, _ := strconv.Atoi(m)
	return num
}

// MQTT 客户端初始化示例
func newMQTTClient() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(Broker)
	opts.SetUsername(MOS_USERNAME)
	opts.SetPassword(MOS_PASSWORD)
	opts.SetClientID(ClientID)
	opts.SetConnectTimeout(5 * time.Second)

	mqttClinet := mqtt.NewClient(opts)
	if token := mqttClinet.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect error: %v", token.Error())
	}
	return mqttClinet
}

// pushAllEnergySensors 推送燃气用量、费用、年度累计三个传感器
func pushAllEnergySensors(httpClinet *http.Client, request_usage, requesrt_cost, last_mon_usage, last_mon_cost, annualUsage float64, usages []MonthlyUsage) error {
	// ################## request data. ##################
	// current month request usage
	log.Println("Tring to push enecle_request_usage")
	if err := pushEnergySensor("sensor.enecle_request_usage", request_usage, "kWh", "energy"); err != nil {
		return err
	}

	// sleep a while
	time.Sleep(800 * time.Millisecond)
	log.Println("Sleep a while...")

	// current month request cost
	log.Println("Tring to push enecle_request_cost")
	if err := pushEnergySensor("sensor.enecle_request_cost", requesrt_cost, "JPY", "monetary"); err != nil {
		return err
	}
	// ################## request data. ##################

	// ################## last month data. ##################
	// current month request usage
	log.Println("Tring to push enecle_last_mon_usage")
	if err := pushEnergySensor("sensor.enecle_last_mon_usage", last_mon_usage, "kWh", "energy"); err != nil {
		return err
	}

	// sleep a while
	time.Sleep(800 * time.Millisecond)
	log.Println("Sleep a while...")

	// current month request cost
	log.Println("Tring to push enecle_last_mon_cost")
	if err := pushEnergySensor("sensor.enecle_last_mon_cost", last_mon_cost, "JPY", "monetary"); err != nil {
		return err
	}

	// ################## last month data. ##################

	// sleep a while
	time.Sleep(800 * time.Millisecond)
	log.Println("Sleep a while...")

	// 年度累计燃气量
	log.Println("Tring to push enecle_annual_usage")
	if err := pushEnergySensor("sensor.enecle_annual_usage", annualUsage, "kWh", "energy"); err != nil {
		return err
	}
	// // 上传到 Home Assistant 统计 API
	// log.Println("Tring to push enecle_usage")
	// err := pushMonthlyUsage(httpClinet, HA_URL, "sensor.enecle_last_mon_usage", usages)
	// if err != nil {
	// 	return err
	// }

	return nil
}

// extractLinks 从 HTML 内容中提取所有 a 标签的文字和链接
func extractLinks(htmlContent string) ([]map[string]string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("解析HTML失败: %v", err)
	}

	var links []map[string]string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var href string
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href = attr.Val
					break
				}
			}
			if href != "" {
				text := extractText(n)
				links = append(links, map[string]string{
					"text": text,
					"href": href,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return links, nil
}

// 提取节点中的纯文本
func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}
	var result string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result += extractText(c)
	}
	return strings.TrimSpace(result)
}
