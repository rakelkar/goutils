package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rakelkar/goutils/pkg/leader"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	zaplog, _ := zap.NewProduction()
	logger := zaplog.Sugar()
	defer logger.Sync() // flushes buffer, if any

	azStorageAccount, err := getStorageAccount()
	if err != nil {
		logger.Error(err, "unable to parse storage account details")
		os.Exit(1)
	}

	stop := make(chan struct{})

	// Start the controller as a separate go routine, if i am leader
	startfunc := func() error {
		// Start the Cmd
		logger.Info("Starting the Cmd.")
		spawnProcess(logger)
		close(stop)
		return nil
	}

	mutex := leader.NewBlobDistributedMutex(logger, azStorageAccount)
	if err := mutex.RunTaskWhenMutexAcquired(ctx, stop, startfunc); err != nil {
		logger.Error(err, "unable to run the manager")
		os.Exit(1)
	}

	logger.Info("Terminating.")
}

func getStorageAccount() (leader.AzureStorageAccountConfiguration, error) {

	accountName := flag.String("a", os.Getenv("ACCOUNT_NAME"), "storage account name")
	containerName := flag.String("c", os.Getenv("CONTAINER_NAME"), "container name")
	accountKey := flag.String("k", os.Getenv("STORAGE_KEY"), "storage account key")
	leaseDurationStr := flag.String("l", os.Getenv("LEASE_DURATION_SECS"), "lease duration")
	renewDurationStr := flag.String("r", os.Getenv("RENEW_DURATION_SECS"), "renew duration")
	acquireDurationStr := flag.String("q", os.Getenv("ACQUIRE_DURATION_SECS"), "acquire poll duration")

	flag.Parse()

	leaseDuration, err := time.ParseDuration(*leaseDurationStr)
	if err != nil {
		return leader.AzureStorageAccountConfiguration{}, fmt.Errorf("config: invalid value %s %v", "leaseDuration", err)
	}

	renewDuration, err := time.ParseDuration(*renewDurationStr)
	if err != nil {
		return leader.AzureStorageAccountConfiguration{}, fmt.Errorf("config: invalid value %s %v", "renewDuration", err)
	}

	acquireDuration, err := time.ParseDuration(*acquireDurationStr)
	if err != nil {
		return leader.AzureStorageAccountConfiguration{}, fmt.Errorf("config: invalid value %s %v", "acquireDuration", err)
	}

	if leaseDuration < renewDuration {
		return leader.AzureStorageAccountConfiguration{}, fmt.Errorf("config: renew duration should be less than lease duration")
	}

	return leader.AzureStorageAccountConfiguration{
		Name:                    *accountName,
		ContainerName:           *containerName,
		AccessKey:               *accountKey,
		LeaseDuration:           leaseDuration,
		RenewIntervalDuration:   renewDuration,
		AcquireIntervalDuration: acquireDuration,
	}, nil
}

func spawnProcess(logger *zap.SugaredLogger) int {
	for i := 0; i < 100; i++ {
		logger.Info("running")
		time.Sleep(1 * time.Second)
	}

	return 0
}
