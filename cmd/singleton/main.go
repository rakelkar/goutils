package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rakelkar/goutils/pkg/leader"
	"go.uber.org/zap"
)

type ParsedOptions struct {
	// Storage config
	StorageConfig leader.AzureStorageAccountConfiguration
	// Command to exeute as singleton
	CommandPath string
	// Arguments of the command
	CommandArgs []string
}

func main() {
	ctx := context.Background()
	zaplog, _ := zap.NewProduction()
	logger := zaplog.Sugar()
	defer logger.Sync() // flushes buffer, if any

	parsedArgs, err := parseArgs()
	if err != nil {
		logger.Error(err, "unable to parse storage account details")
		os.Exit(1)
	}

	stop := make(chan struct{})

	// Start the controller as a separate go routine, if i am leader
	startfunc := func() error {
		// Start the Cmd
		logger.Info("Starting the Cmd.")
		spawnProcess(logger, parsedArgs.CommandPath, parsedArgs.CommandArgs)
		close(stop)
		return nil
	}

	mutex := leader.NewBlobDistributedMutex(logger, parsedArgs.StorageConfig)
	if err := mutex.RunTaskWhenMutexAcquired(ctx, stop, startfunc); err != nil {
		logger.Error(err, "unable to run the manager")
		os.Exit(1)
	}

	logger.Info("Terminating.")
}

func parseArgs() (ParsedOptions, error) {

	parsedArgs := ParsedOptions{}

	// storage options
	accountName := flag.String("a", os.Getenv("SINGLETON_ACCOUNT_NAME"), "storage account name")
	containerName := flag.String("c", os.Getenv("SINGLETON_CONTAINER_NAME"), "container name")
	accountKey := flag.String("k", os.Getenv("SINGLETON_STORAGE_KEY"), "storage account key")
	leaseDurationStr := flag.String("l", os.Getenv("SINGLETON_LEASE_DURATION_SECS"), "lease duration")
	renewDurationStr := flag.String("r", os.Getenv("SINGLETON_RENEW_DURATION_SECS"), "renew duration")
	acquireDurationStr := flag.String("q", os.Getenv("SINGLETON_ACQUIRE_DURATION_SECS"), "acquire poll duration")

	// command options
	commandPath := flag.String("cmd", os.Getenv("SINGLETON_CMD"), "command to execute as singleton")
	commandArgsStr := flag.String("args", os.Getenv("SINGLETON_CMD_ARGS"), "commmand arguments list")

	flag.Parse()

	leaseDuration, err := time.ParseDuration(*leaseDurationStr)
	if err != nil {
		return ParsedOptions{}, fmt.Errorf("config: invalid value %s %v", "leaseDuration", err)
	}

	renewDuration, err := time.ParseDuration(*renewDurationStr)
	if err != nil {
		return ParsedOptions{}, fmt.Errorf("config: invalid value %s %v", "renewDuration", err)
	}

	acquireDuration, err := time.ParseDuration(*acquireDurationStr)
	if err != nil {
		return ParsedOptions{}, fmt.Errorf("config: invalid value %s %v", "acquireDuration", err)
	}

	if leaseDuration < renewDuration {
		return ParsedOptions{}, fmt.Errorf("config: renew duration should be less than lease duration")
	}

	parsedArgs.CommandPath = *commandPath
	parsedArgs.CommandArgs = strings.Split(*commandArgsStr, " ")
	parsedArgs.StorageConfig = leader.AzureStorageAccountConfiguration{
		Name:                    *accountName,
		ContainerName:           *containerName,
		AccessKey:               *accountKey,
		LeaseDuration:           leaseDuration,
		RenewIntervalDuration:   renewDuration,
		AcquireIntervalDuration: acquireDuration,
	}

	return parsedArgs, nil
}

func spawnProcess(logger *zap.SugaredLogger, cmdPath string, cmdArgs []string) {
	logger.Infof("Starting Cmd: [%s] with arguments [%s]", cmdPath, cmdArgs)
	cmd := exec.Command(cmdPath, cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}
	logger.Info("Cmd completed.")
}
func idleLoop(logger *zap.SugaredLogger) int {
	for i := 0; i < 100; i++ {
		logger.Info("running")
		time.Sleep(1 * time.Second)
	}

	return 0
}
