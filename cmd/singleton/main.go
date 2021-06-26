package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
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
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"stdout"}
	zapLogger, err := config.Build()
	logger := zapLogger.Sugar()

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
		spawnProcess(logger, stop, parsedArgs.CommandPath, parsedArgs.CommandArgs)
		if _, ok := <- stop; ok {
			close(stop)
		}
		return nil
	}

	mutex := leader.NewBlobDistributedMutex(logger, parsedArgs.StorageConfig)
	if err := mutex.RunTaskWhenMutexAcquired(ctx, stop, startfunc); err != nil {
		logger.Error(err, "unable to run singleton locker")
		os.Exit(1)
	}

	logger.Info("Terminating.")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func parseArgs() (ParsedOptions, error) {

	parsedArgs := ParsedOptions{}

	// storage options
	accountName := flag.String("a", os.Getenv("SINGLETON_ACCOUNT_NAME"), "storage account name")
	containerName := flag.String("c", os.Getenv("SINGLETON_CONTAINER_NAME"), "container name")
	accountKey := flag.String("k", os.Getenv("SINGLETON_STORAGE_KEY"), "storage account key")
	leaseDurationStr := flag.String("l", getEnv("SINGLETON_LEASE_DURATION", "30s"), "lease duration e.g. 30s")
	renewDurationStr := flag.String("r", getEnv("SINGLETON_RENEW_DURATION", "5s"), "renew duration")
	acquireDurationStr := flag.String("q", getEnv("SINGLETON_ACQUIRE_DURATION", "15s"), "acquire poll duration")

	// command options
	commandPath := flag.String("cmd", os.Getenv("SINGLETON_CMD"), "command to execute as singleton")
	commandArgsStr := flag.String("args", os.Getenv("SINGLETON_CMD_ARGS"), "commmand arguments list")
	commandArgsSeparator := flag.String("sep", getEnv("SINGLETON_CMD_ARG_SEPARATOR", " "), "commmand arguments list separator (defaul to space)")
	isTestMode := flag.Bool("t", false, "run in test mode (no validation)")

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

	if leaseDuration < renewDuration && !(*isTestMode) {
		return ParsedOptions{}, fmt.Errorf("config: renew duration should be less than lease duration")
	}

	parsedArgs.CommandPath = *commandPath
	parsedArgs.CommandArgs = strings.Split(*commandArgsStr, *commandArgsSeparator)
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

func spawnProcess(logger *zap.SugaredLogger, stop chan struct{}, cmdPath string, cmdArgs []string) {
	logger.Infof("Starting Cmd: [%s] with arguments [%s]", cmdPath, cmdArgs)
	cmd := exec.Command(cmdPath, cmdArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		logger.Fatal("Cmd failed to start with error: %v", err)
		return
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case _, ok := <-stop:
		if !ok {
			logger.Warn("stop closed (process terminated?)")
			return
		}

		logger.Info("stop triggered.")
		if err := cmd.Process.Kill(); err != nil {
			logger.Fatal("failed to kill process", err)
		}
		logger.Info("process killed on stop trigger")

	case err := <-done:
		if err != nil {
			logger.Fatal("process finished with error", err)
		}
		logger.Info("process finished successfully")
	}	
}

func idleLoop(logger *zap.SugaredLogger) int {
	for i := 0; i < 100; i++ {
		logger.Info("running")
		time.Sleep(1 * time.Second)
	}

	return 0
}
