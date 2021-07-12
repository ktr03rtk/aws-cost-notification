package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/pkg/errors"
)

func handler() error {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return errors.Wrap(err, "failed to initialize cost explorer client")
	}

	explorer := costexplorer.NewFromConfig(cfg)

	getCostAndUsageInput := &costexplorer.GetCostAndUsageInput{
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		TimePeriod:  &types.DateInterval{Start: aws.String("2021-07-01"), End: aws.String("2021-07-11")},
		GroupBy:     []types.GroupDefinition{{Key: aws.String("SERVICE"), Type: "DIMENSION"}},
	}

	output, _ := explorer.GetCostAndUsage(context.TODO(), getCostAndUsageInput)
	if err != nil {
		return errors.Wrap(err, "failed to get cost and usage")
	}

	bytes, err := json.Marshal(output)
	if err != nil {
		return errors.Wrap(err, "failed to json marshal")
	}

	fmt.Printf("--------------- %+v\n", string(bytes))

	return nil

}

func main() {
	lambda.Start(handler)
}
