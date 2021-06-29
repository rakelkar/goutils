# expected to die gracefully after getting lock and executing delay
./singleton -q 30s -r 5s -l 10s -cmd "./delay.sh" -args "primary running" 

ps -ef | grep singleton | grep -v grep