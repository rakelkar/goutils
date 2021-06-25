# primary acquires lock for 1m, but renews only after 5m (so will loose lock)
./singleton -q 30s -r 51s -l 30s -t -cmd "./sleep.sh" -args "primary b" &

# sleep to let primary acquire lock
sleep 15

# secondary acquires lock for 30s and renews in 15s so should win lock
./singleton -q 5s -r 30s -l 60s -cmd "./sleep.sh" -args "secondary b"