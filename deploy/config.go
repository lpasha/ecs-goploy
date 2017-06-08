package deploy

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/defaults"
)

// newConfig returns a new aws config
func newConfig(profile string, region string) *aws.Config {
	defaultConfig := defaults.Get().Config
	cred := newCredentials(getenv(profile, "AWS_DEFAULT_PROFILE"), getenv(region, "AWS_DEFAULT_REGION"))
	return defaultConfig.WithCredentials(cred).WithRegion(getenv(region, "AWS_DEFAULT_REGION"))
}

func newCredentials(profile string, region string) *credentials.Credentials {
	// temporary config to resolve RemoteCredProvider
	tmpConfig := defaults.Get().Config.WithRegion(region)
	tmpHandlers := defaults.Handlers()

	return credentials.NewChainCredentials(
		[]credentials.Provider{
			// Read profile before environment variables
			&credentials.SharedCredentialsProvider{
				Profile: profile,
			},
			&credentials.EnvProvider{},
			// for IAM Task Role (ECS) and IAM Role
			defaults.RemoteCredProvider(*tmpConfig, tmpHandlers),
		})
}

func getenv(value, key string) string {
	if len(value) == 0 {
		return os.Getenv(key)
	}
	return value
}
