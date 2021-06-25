ps -ef | grep -v grep | grep singleton | cut -f2 -d' ' | xargs kill -9
ps -ef | grep -v grep | grep "sleep.sh" | cut -f2 -d' ' | xargs kill -9
ps -ef | grep singleton
