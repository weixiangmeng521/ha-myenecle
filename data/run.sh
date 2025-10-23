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
    username=$(bashio::config 'username')
    password=$(bashio::config 'password')
    mosquitto_username="$(bashio::services 'mosquitto' 'username')"
    mosquitto_password="$(bashio::services 'mosquitto' 'password')"

    # set env
    export USERNAME="$username"
    export PASSWORD="$password"
    export MOSQUITTO_USERNAME="$mosquitto_username"
    export MOSQUITTO_PASSWORD="$mosquitto_password"

    echo "export mosquitto_username=\"${mosquitto_username}\""
    echo "export mosquitto_password=\"${mosquitto_password}\""

    # excute
    /usr/bin/myenecle -u "$username" -p "$password" -p "$mosquitto_username" -p "$mosquitto_password"
    sleep 3600
done
