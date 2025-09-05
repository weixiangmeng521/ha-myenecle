#!/usr/bin/with-contenv bashio
set -e

username=$(bashio::config 'username')
password=$(bashio::config 'password')
haToken=$(bashio::config 'long_live_token')

# set env
export USERNAME="$username"
export PASSWORD="$password"
export HA_TOKEN="$haToken"

# excute
/usr/bin/myenecle -u "$username" -p "$password" -t "$haToken"
