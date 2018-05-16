package aws_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/lambda"

	"github.com/gruntwork-io/terratest/modules/terraform"

	"github.com/segmentio/ebs-backup/internal/handler"
)

func TestEBSBackup(t *testing.T) {
	sess := getAWSSession()

	defer terraform.Destroy(t, tfOpts())
	terraform.InitAndApply(t, tfOpts())

	backupFunctionName := terraform.Output(t, tfOpts(), "backup_function_name")
	backupFunctionARN := terraform.Output(t, tfOpts(), "backup_function_arn")

	tests := map[string]func(t *testing.T){
		"CloudWatch Scheduled Event created": testScheduledEvent(sess, backupFunctionARN),
		"Create snapshot":                    testCreateSnapshot(sess, backupFunctionName),
	}

	t.Run("AWS", func(t *testing.T) {
		for name, test := range tests {
			t.Run(name, test)
		}
	})
}

func testScheduledEvent(sess client.ConfigProvider, lambdaARN string) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		client := cloudwatchevents.New(sess)
		output, err := client.ListRuleNamesByTarget(
			&cloudwatchevents.ListRuleNamesByTargetInput{
				TargetArn: aws.String(lambdaARN),
			},
		)
		if err != nil {
			t.Fatal(err)
		}

		for _, name := range output.RuleNames {
			output, err := client.DescribeRule(
				&cloudwatchevents.DescribeRuleInput{
					Name: name,
				},
			)
			if err != nil {
				t.Fatal(err)
			}

			if aws.StringValue(output.ScheduleExpression) != "" {
				return
			}
		}
		t.Error("Could not find scheduled backup CloudWatch event")
	}
}

func testCreateSnapshot(sess client.ConfigProvider, functionName string) func(*testing.T) {
	return func(t *testing.T) {
		t.Parallel()

		lambdaClient := lambda.New(sess)

		output, err := lambdaClient.Invoke(&lambda.InvokeInput{
			FunctionName:   aws.String(functionName),
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
			defer deleteSnapshot(sess, resp[i].SnapshotID)
		}

		t.Run("Single snapshot created", func(t *testing.T) {
			if len(resp) != 1 {
				t.Errorf("Expected 1 snapshot created, actual %d", len(resp))
			}
		})

		ec2Client := ec2.New(sess)

		t.Run("Snapshot exists", func(t *testing.T) {
			resp, err := ec2Client.DescribeSnapshots(
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
					"Name":    nameTag(),
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
}

func deleteSnapshot(sess client.ConfigProvider, snapshotID string) {
	ec2Client := ec2.New(sess)
	ec2Client.DeleteSnapshot(&ec2.DeleteSnapshotInput{
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

func nameTag() string {
	return "EBSBackupTest-" + envMust("CIRCLE_WORKFLOW_ID")
}

func buildUser() string {
	return envMust("CIRCLE_USERNAME", "LOGNAME")
}

func tfOpts() *terraform.Options {
	return &terraform.Options{
		Vars: map[string]interface{}{
			"build_username":   buildUser(),
			"workflow_id":      envMust("CIRCLE_WORKFLOW_ID"),
			"lambda_s3_bucket": envMust("LAMBDA_S3_BUCKET"),
			"lambda_s3_key":    envMust("LAMBDA_S3_KEY"),
			"name":             nameTag(),

			"assume_role_arn": os.Getenv("ASSUME_ROLE_ARN"),
		},
	}
}

func getAWSSession() *session.Session {
	var region string

	sess := session.Must(session.NewSession())

	// Detect region automatically if possible
	client := ec2metadata.New(sess)
	doc, err := client.GetInstanceIdentityDocument()
	if err == nil {
		sess.Config.Region = aws.String(doc.Region)
	}

	assumeRoleARN := os.Getenv("ASSUME_ROLE_ARN")
	if assumeRoleARN != "" {
		// Replace session with one configured to assume role
		sess = session.Must(
			session.NewSession(
				aws.NewConfig().WithCredentials(
					stscreds.NewCredentials(sess, assumeRoleARN),
				).WithRegion(region),
			),
		)
	}

	return sess
}
