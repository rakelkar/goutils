package leader

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type AzureStorageAccountConfiguration struct {
	// Name of the azure blob storage account
	Name string
	// Name of the blob container on which we lease
	ContainerName string
	// AccessKey for the blob storage account
	AccessKey string
	// RenewIntervalDuration in which we renew the lease on the blob container
	RenewIntervalDuration time.Duration
	//AcquireIntervalDuration in which we again try to aquire lease on the blob container
	AcquireIntervalDuration time.Duration
	//LeaseDuration is the duration of the lease
	LeaseDuration time.Duration
}

// BlobDistributedMutex is responsible of executing a function after taking a lease on a blob
type BlobDistributedMutex struct {
	logger                    *zap.SugaredLogger
	azureStorageAccountConfig AzureStorageAccountConfiguration
	leaseManager              *BlobLeaseManager
}

// NewBlobDistributedMutex returns a new instance of BlobDistributedMutex
func NewBlobDistributedMutex(log *zap.SugaredLogger, azureStorageConfig AzureStorageAccountConfiguration) *BlobDistributedMutex {
	return &BlobDistributedMutex{
		azureStorageAccountConfig: azureStorageConfig,
		logger:                    log,
		leaseManager:              &BlobLeaseManager{logger: log}}
}

// RunTaskWhenMutexAcquired execute the given func after a blob lease is acquired
func (bdm *BlobDistributedMutex) RunTaskWhenMutexAcquired(ctx context.Context, stp chan struct{}, taskToRunWhenLeaseAcquired func() error) error {

	bdm.leaseManager.Init(ctx, bdm.azureStorageAccountConfig.Name, bdm.azureStorageAccountConfig.AccessKey, bdm.azureStorageAccountConfig.ContainerName)

	// This is a blocking call till we acquire the lease.
	leaseID := bdm.tryAcquireLeaseOrWait(ctx)
	defer bdm.leaseManager.ReleaseLease(ctx, leaseID)

	// Run  renew routine
	go bdm.keepRenewingLease(ctx, leaseID, stp)

	// Call main function. This is blocking call
	return taskToRunWhenLeaseAcquired()
}

// tryAcquireLeaseOrWait will only returns when it was able to acquire the lease
func (bdm *BlobDistributedMutex) tryAcquireLeaseOrWait(ctx context.Context) string {
	for {
		bdm.logger.Infof("Trying to acquire the lease on %s, %s", bdm.azureStorageAccountConfig.Name, bdm.azureStorageAccountConfig.ContainerName)
		leaseID, err := bdm.leaseManager.AcquireLease(ctx, bdm.azureStorageAccountConfig.LeaseDuration)
		if err == nil {
			return leaseID
		}

		if bdm.leaseManager.checkIfLeaseAlreadyExists(err) {
			bdm.logger.Infof("LeaseAlreadyPresent on %s,%s will try again in %v", bdm.azureStorageAccountConfig.Name, bdm.azureStorageAccountConfig.ContainerName, bdm.azureStorageAccountConfig.AcquireIntervalDuration)

		} else {
			bdm.logger.Warnf("Failed to acquire the lease on %s,%s will try again in %v, %v", bdm.azureStorageAccountConfig.Name, bdm.azureStorageAccountConfig.ContainerName, bdm.azureStorageAccountConfig.AcquireIntervalDuration, err)

		}
		time.Sleep(bdm.azureStorageAccountConfig.AcquireIntervalDuration)
	}
}

// keepRenewingLease keeps on renewing the lease, if any error send message to stop channel
func (bdm *BlobDistributedMutex) keepRenewingLease(ctx context.Context, leaseID string, stp chan struct{}) {

	for {
		bdm.logger.Infof("Trying to renew in %v the lease on %s, %s", bdm.azureStorageAccountConfig.RenewIntervalDuration, bdm.azureStorageAccountConfig.Name, bdm.azureStorageAccountConfig.ContainerName)
		renewed, err := bdm.leaseManager.RenewLease(ctx, leaseID)
		if !renewed {
			bdm.logger.Warnf("Failed to renew the lease on %s,%s, %v", bdm.azureStorageAccountConfig.Name, bdm.azureStorageAccountConfig.ContainerName, err)
			// Terminate process
			stp <- struct{}{}
			return
		}
		time.Sleep(bdm.azureStorageAccountConfig.RenewIntervalDuration)
	}
}
