package waiter

import (
	"time"

	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	SubscriptionPendingConfirmationTimeout = 2 * time.Minute
	SubscriptionDeleteTimeout              = 2 * time.Minute
)

func SubscriptionConfirmed(conn *sns.SNS, id, expectedValue string, timeout time.Duration) (*sns.GetSubscriptionAttributesOutput, error) {
	stateConf := &resource.StateChangeConf{
		Target:  []string{expectedValue},
		Refresh: SubscriptionPendingConfirmation(conn, id),
		Timeout: timeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*sns.GetSubscriptionAttributesOutput); ok {
		return output, err
	}

	return nil, err
}

func SubscriptionDeleted(conn *sns.SNS, id string) (*sns.GetSubscriptionAttributesOutput, error) {
	stateConf := &resource.StateChangeConf{
		Pending: []string{"false", "true"},
		Target:  []string{},
		Refresh: SubscriptionPendingConfirmation(conn, id),
		Timeout: SubscriptionDeleteTimeout,
	}

	outputRaw, err := stateConf.WaitForState()

	if output, ok := outputRaw.(*sns.GetSubscriptionAttributesOutput); ok {
		return output, err
	}

	return nil, err
}
