package leader

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"go.uber.org/zap"
)

// BlobLeaseManager is use to get a lease on a blob container
type BlobLeaseManager struct {
	logger         *zap.SugaredLogger
	leaseContainer azblob.ContainerURL
}

// Init initialize the BlobLeaseManager
func (lm *BlobLeaseManager) Init(ctx context.Context, accountName string, accountKey string, containerName string) error {
	if len(accountName) == 0 || len(accountKey) == 0 {
		lm.logger.Error("Either the AZURE_STORAGE_ACCOUNT or AZURE_STORAGE_ACCESS_KEY environment variable is not set")
	}

	// Create a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		lm.logger.Error("Invalid credentials with error: ", err)
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})

	// From the Azure portal, get your storage account blob service URL endpoint.
	URL, _ := url.Parse(
		fmt.Sprintf("https://%s.blob.core.windows.net/%s", accountName, containerName))

	// Create a ContainerURL object that wraps the container URL and a request
	// pipeline to make requests.
	containerURL := azblob.NewContainerURL(*URL, p)

	// Create the container
	lm.logger.Infof("Creating a container named %s\n", containerName)
	_, err = containerURL.Create(ctx, azblob.Metadata{}, azblob.PublicAccessNone)
	if err == nil || lm.checkIfContainerExists(err) {
		lm.leaseContainer = containerURL
		return nil
	}

	return err
}

//ReleaseLease removes the active lease on the container
func (lm *BlobLeaseManager) ReleaseLease(ctx context.Context, leaseID string) (bool, error) {
	_, err := lm.leaseContainer.ReleaseLease(ctx, leaseID, azblob.ModifiedAccessConditions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

//AcquireLease create a lease on the given blob container
func (lm *BlobLeaseManager) AcquireLease(ctx context.Context, leaseDuration time.Duration) (string, error) {
	response, err := lm.leaseContainer.AcquireLease(ctx, "", int32(leaseDuration.Seconds()), azblob.ModifiedAccessConditions{})
	if err != nil {
		return "", err
	}
	return response.LeaseID(), nil
}

//RenewLease renew the given lease container
func (lm *BlobLeaseManager) RenewLease(ctx context.Context, leaseID string) (bool, error) {
	_, err := lm.leaseContainer.RenewLease(ctx, leaseID, azblob.ModifiedAccessConditions{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (lm *BlobLeaseManager) checkIfContainerExists(err error) bool {
	if err != nil {
		if serr, ok := err.(azblob.StorageError); ok { // This error is a Service-specific
			switch serr.ServiceCode() { // Compare serviceCode to ServiceCodeXxx constants
			case azblob.ServiceCodeContainerAlreadyExists:
				lm.logger.Infof("Received 409. Container already exists")
				return true
			}
		}
		return false
	}
	return true
}

func (lm *BlobLeaseManager) checkIfLeaseAlreadyExists(err error) bool {
	if err != nil {
		if serr, ok := err.(azblob.StorageError); ok { // This error is a Service-specific
			switch serr.ServiceCode() { // Compare serviceCode to ServiceCodeXxx constants
			case azblob.ServiceCodeLeaseAlreadyPresent:
				return true
			}
		}
		return false
	}
	return true
}
