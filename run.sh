#!/usr/bin/with-contenv bashio
set -e

while true; do
    username=$(bashio::config 'username')
    password=$(bashio::config 'password')
    haToken=$(bashio::config 'long_live_token')

    USERNAME="$username" PASSWORD="$password" HA_TOKEN="$haToken" /enecle-linux-arm64
    sleep 1800
done
