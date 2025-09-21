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
    haToken=$(bashio::config 'long_live_token')

    # set env
    export USERNAME="$username"
    export PASSWORD="$password"

    # excute
    /usr/bin/myenecle -u "$username" -p "$password"
    sleep 3600
done
