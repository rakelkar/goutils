# Singleton Spawner
Spawns a singleton process by taking a distributed lock on an Azure Blob

Example usage:
```
./singleton -a storageaccount -c storagecontainername -k storageaccountkey -r 10s -q 10s -l 30s -cmd /path/command -args "--some arg --someOther arg"
```