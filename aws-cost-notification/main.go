package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/pkg/errors"
)

func handler() error {
	const shortForm = "2006-01-02"

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return errors.Wrap(err, "failed to initialize cost explorer client")
	}

	explorer := costexplorer.NewFromConfig(cfg)

	var startDay, endDay string
	t := time.Now()

	// explore cost of previous month at the beginning of month
	if t.Day() == 1 {
		startDay = t.AddDate(0, -1, 0).Format(shortForm)
		endDay = t.AddDate(0, 0, -1).Format(shortForm)
	} else {
		startDay = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).Format(shortForm)
		endDay = t.Format(shortForm)
	}

	getCostAndUsageInput := &costexplorer.GetCostAndUsageInput{
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		TimePeriod:  &types.DateInterval{Start: aws.String(startDay), End: aws.String(endDay)},
		GroupBy:     []types.GroupDefinition{{Key: aws.String("SERVICE"), Type: "DIMENSION"}},
	}

	output, err := explorer.GetCostAndUsage(context.TODO(), getCostAndUsageInput)
	if err != nil {
		return errors.Wrap(err, "failed to get cost and usage")
	}

	total := new(big.Float)
	for _, v := range output.ResultsByTime[0].Groups {
		f, ok := new(big.Float).SetString(*v.Metrics["UnblendedCost"].Amount)
		if !ok {
			return errors.New("failed to parse big float")
		}

		total = new(big.Float).Add(total, f)

	}

	bytes, err := json.Marshal(output.ResultsByTime[0].Groups)
	if err != nil {
		return errors.Wrap(err, "failed to json marshal")
	}

	fmt.Printf("--------------- %+v\n", string(bytes))
	fmt.Printf("--------------- %+v\n", output.ResultsByTime[0].TimePeriod)
	fmt.Printf("--------------- %+v\n", total.String())

	return nil

}

func main() {
	lambda.Start(handler)
}
