#!/usr/bin/with-contenv bashio
# ==============================================================================
# Start the example service
# s6-overlay docs: https://github.com/just-containers/s6-overlay
# ==============================================================================

# Declare variables
declare message
declare password
declare HA_TOKEN
declare cookie_file

## Get the 'message' key from the user config options.
username=$(bashio::config 'username')
password=$(bashio::config 'password')
HA_TOKEN=$(bashio::config 'long_live_token')

# Temporary file to store cookies
cookie_file="/tmp/enecle_cookies.txt"

# Your Home Assistant URL
HA_URL="http://homeassistant:8123"

# set -e

# 定义兼容日志函数
log_info() {
    bashio::log.info "$1"
}


# 如果没有 bashio，使用环境变量
log_info "Starting Enecle fetch with username: ${username}"

# 首先加载页面，获取页面中的TOKEN，从name=__RequestVerificationToken的input标签里获取value，设置成变量token
login_page=$(curl -s -c "$cookie_file" "https://myenecle.com/Login")
token=$(echo "$login_page" | sed -n 's/.*name="__RequestVerificationToken"[^>]*value="\([^"]*\)".*/\1/p')
log_info "Fetched token: $token"

# 以下三个字段，拿来请求 ”https://myenecle.com/Login“ 获取 cookie
# __RequestVerificationToken 是 token
# MailAddress 是 username
# password 是 password

login_response=$(curl -s -b "$cookie_file" -c "$cookie_file" \
  -X POST "https://myenecle.com/Login" \
  -d "__RequestVerificationToken=${token}&MailAddress=${username}&Password=${password}")
# log_info "Login response: $login_response"

# 假设 login_response 变量里有 HTML 内容
error_msg=$(echo "$login_response" | \
  sed -n 's/.*<div class="validation-summary-errors"[^>]*>.*<ul><li>\(.*\)<\/li>.*/\1/p')

# 如果是 <ul class="errmsg"> 结构
if [ -z "$error_msg" ]; then
  error_msg=$(echo "$login_response" | \
    sed -n 's/.*<ul class="errmsg">.*<li>.*<ul><li>\(.*\)<\/li>.*/\1/p')
fi
# 假设 error_msg 变量里是 HTML 实体编码
decoded=$(echo "$error_msg" | perl -CSD -MHTML::Entities -pe 'decode_entities($_)')
log_info "Error: $decoded"


# 查看保存的 cookies
cookies=$(cat $cookie_file)
log_info "$cookies"

# 使用登录后的 cookies 访问 MyPageTop
mypage_response=$(curl -s -b "$cookie_file" \
  -H "User-Agent: Mozilla/5.0" \
  -H "Referer: https://myenecle.com/Login" \
  "https://myenecle.com/MyPage/MyPageTop")
  
log_info "Fetched MyPageTop..."

usage=$(echo "$mypage_response" | sed -n 's/.*<li><span>&#x3054;&#x4F7F;&#x7528;&#x91CF;<\/span><span><em>\([^<]*\)<\/em>m3<\/span><\/li>.*/\1/p')
log_info "gas usage: $usage"


# 判断是否有错误消息
if [ -n "$decoded" ]; then
    # 有错误
    curl -X POST "$HA_URL/api/states/sensor.enecle_usage" \
      -H "Authorization: Bearer $HA_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"state\": \"error\", \"attributes\": {\"friendly_name\": \"Enecle usage\", \"error_msg\": \"$decoded\"}}"
else
    # 正常推送
    curl -X POST "$HA_URL/api/states/sensor.enecle_usage" \
      -H "Authorization: Bearer $HA_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"state\": \"$usage\", \"attributes\": {\"unit_of_measurement\": \"m3\", \"friendly_name\": \"Enecle usage\"}}"
fi
