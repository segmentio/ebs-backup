package aws_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/gruntwork-io/terratest/modules/terraform"

	"github.com/segmentio/ebs-backup/internal/handler"
)

var (
	sess client.ConfigProvider
)

type ctxKey int

const (
	awsSession ctxKey = iota
)

func getSessionCtx(ctx context.Context) context.Context {
	sess = session.Must(session.NewSession())
	assumeRoleARN := os.Getenv("ASSUME_ROLE_ARN")
	if assumeRoleARN != "" {
		// Replace session with one configured to assume role
		sess = session.Must(
			session.NewSession(
				aws.NewConfig().WithCredentials(
					stscreds.NewCredentials(sess, assumeRoleARN),
				),
			),
		)
	}
	return context.WithValue(ctx, awsSession, sess)
}

func TestCreateSnapshot(t *testing.T) {
	defer terraform.Destroy(t, tfOpts())
	terraform.InitAndApply(t, tfOpts())

	ctx := getSessionCtx(context.Background())

	lambdaClient := lambda.New(ctx.Value(awsSession).(client.ConfigProvider))

	output, err := lambdaClient.Invoke(&lambda.InvokeInput{
		FunctionName:   aws.String(lambdaFunctionName()),
		InvocationType: aws.String("RequestResponse"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if output.FunctionError != nil {
		t.Fatalf("Lambda returned function error: %s", aws.StringValue(output.FunctionError))
	}

	var resp handler.Response
	if err := json.Unmarshal(output.Payload, &resp); err != nil {
		t.Fatal(err)
	}
	for i := range resp {
		defer deleteSnapshot(ctx, resp[i].SnapshotID)
	}

	t.Run("Single snapshot created", func(t *testing.T) {
		if len(resp) != 1 {
			t.Errorf("Expected 1 snapshot created, actual %d", len(resp))
		}
	})

	ec2Client := ec2.New(ctx.Value(awsSession).(client.ConfigProvider))

	t.Run("Snapshot exists", func(t *testing.T) {
		resp, err := ec2Client.DescribeSnapshotsWithContext(ctx,
			&ec2.DescribeSnapshotsInput{
				SnapshotIds: aws.StringSlice([]string{resp[0].SnapshotID}),
			},
		)
		if err != nil {
			t.Errorf("DescribeSnapshots: %v", err)
		}
		if len(resp.Snapshots) != 1 {
			t.Errorf("Expected 1 snapshot, actual %d", len(resp.Snapshots))
		}

		t.Run("Snapshot has proper tags", func(t *testing.T) {
			tags := map[string]string{
				"Name":    lambdaFunctionName(),
				"Creator": buildUser(),
			}
			for k, v := range tags {
				var found bool
				for _, tag := range resp.Snapshots[0].Tags {
					if aws.StringValue(tag.Key) == k && aws.StringValue(tag.Value) == v {
						found = true
					}
				}
				if !found {
					t.Errorf("Snapshot missing expected tag %s: %s", k, v)
				}
			}
		})
	})
}

func deleteSnapshot(ctx context.Context, snapshotID string) {
	ec2Client := ec2.New(ctx.Value(awsSession).(client.ConfigProvider))
	ec2Client.DeleteSnapshotWithContext(ctx, &ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(snapshotID),
	})
}

func envMust(keys ...string) (val string) {
	for _, key := range keys {
		if val = os.Getenv(key); val != "" {
			return val
		}
	}
	panic(fmt.Sprintf("%s must be set", strings.Join(keys, " or ")))
}

func lambdaFunctionName() string {
	return "EBSBackupTest-" + envMust("CIRCLE_WORKFLOW_ID")
}

func buildUser() string {
	return envMust("CIRCLE_USERNAME", "LOGNAME")
}

func tfOpts() *terraform.Options {
	return &terraform.Options{
		Vars: map[string]interface{}{
			"build_username":       buildUser(),
			"workflow_id":          envMust("CIRCLE_WORKFLOW_ID"),
			"lambda_s3_bucket":     envMust("LAMBDA_S3_BUCKET"),
			"lambda_s3_key":        envMust("LAMBDA_S3_KEY"),
			"lambda_function_name": lambdaFunctionName(),
			"name":                 lambdaFunctionName(),

			"assume_role_arn": os.Getenv("ASSUME_ROLE_ARN"),
		},
	}
}
