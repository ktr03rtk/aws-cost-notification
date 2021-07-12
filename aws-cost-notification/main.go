package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/pkg/errors"
)

type params struct {
	Text      string `json:"text"`
	Username  string `json:"username"`
	IconEmoji string `json:"icon_emoji"`
	IconURL   string `json:"icon_url"`
	Channel   string `json:"channel"`
}

func handler() error {
	const shortForm = "2006-01-02"

	slackURL, ok := os.LookupEnv("SLACK_WEBHOOK_URL")
	if !ok {
		return errors.New("env SLACK_WEBHOOK_URL is not found")
	}

	slackChannel, ok := os.LookupEnv("SLACK_CHANNEL")
	if !ok {
		return errors.New("env SLACK_CHANNEL is not found")
	}

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

	p := params{
		Text:     string(bytes),
		Username: "Cost Explorer",
		Channel:  slackChannel,
	}

	params, err := json.Marshal(p)
	if err != nil {
		return errors.Wrap(err, "failed to json marshal")
	}

	resp, err := http.PostForm(
		slackURL,
		url.Values{"payload": {string(params)}},
	)
	if err != nil {
		return errors.Wrap(err, "failed to send Slack message")
	}

	defer resp.Body.Close()

	return nil
}

func main() {
	lambda.Start(handler)
}
