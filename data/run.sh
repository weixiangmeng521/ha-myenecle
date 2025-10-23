#!/usr/bin/with-contenv bashio
set -e

#  ____    __  __  ____    ____     __       ____      
# /\  _`\ /\ \/\ \/\  _`\ /\  _`\  /\ \     /\  _`\    
# \ \ \L\_\ \ `\\ \ \ \L\_\ \ \/\_\\ \ \    \ \ \L\_\  
#  \ \  _\L\ \ , ` \ \  _\L\ \ \/_/_\ \ \  __\ \  _\L  
#   \ \ \L\ \ \ \`\ \ \ \L\ \ \ \L\ \\ \ \L\ \\ \ \L\ \
#    \ \____/\ \_\ \_\ \____/\ \____/ \ \____/ \ \____/
#     \/___/  \/_/\/_/\/___/  \/___/   \/___/   \/___/ 
     
cat << "EOF"
 ____    __  __  ____    ____     __       ____      
/\  _`\ /\ \/\ \/\  _`\ /\  _`\  /\ \     /\  _`\    
\ \ \L\_\ \ `\\ \ \ \L\_\ \ \/\_\\ \ \    \ \ \L\_\  
 \ \  _\L\ \ , ` \ \  _\L\ \ \/_/_\ \ \  __\ \  _\L  
  \ \ \L\ \ \ \`\ \ \ \L\ \ \ \L\ \\ \ \L\ \\ \ \L\ \
   \ \____/\ \_\ \_\ \____/\ \____/ \ \____/ \ \____/
    \/___/  \/_/\/_/\/___/  \/___/   \/___/   \/___/ 
EOF


while true; do
    CONFIG_USERNAME=$(bashio::config 'username')
    CONFIG_PASSWORD=$(bashio::config 'password')
    TEPCO2MQTT_CONFIG_MQTT_USERNAME="$(bashio::services 'mqtt' 'username')"
    TEPCO2MQTT_CONFIG_MQTT_PASSWORD="$(bashio::services 'mqtt' 'password')"    

    echo "======================================================="
    echo "MQTT username: ${TEPCO2MQTT_CONFIG_MQTT_USERNAME}"
    echo "MQTT password length: ${#TEPCO2MQTT_CONFIG_MQTT_PASSWORD}"
    echo "======================================================="

    # excute
    /usr/bin/myenecle -u "$CONFIG_USERNAME" -p "$CONFIG_PASSWORD" -p "$TEPCO2MQTT_CONFIG_MQTT_USERNAME" -p "$TEPCO2MQTT_CONFIG_MQTT_PASSWORD"
    sleep 3600
done
