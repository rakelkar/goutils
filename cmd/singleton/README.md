# Singleton Spawner
Spawns a singleton process by taking a distributed lock on an Azure Blob

Example usage:
```bash
./singleton \
  -a storageaccount \
  -c storagecontainername \
  -k storageaccountkey \
  -r 10s \
  -q 10s \
  -l 30s \
  -cmd "./sleep.sh" \       # full path to command to execute as singleton
  -args "a b"    # arugments (get split on space) e.g. "--option1 value1 --option2 value2"
```

Also supports environment variables instead of args e.g. `SINGLETON_ACCOUNT_NAME`

