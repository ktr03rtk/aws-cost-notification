package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/pkg/errors"
)

var (
	slackURL     string
	slackChannel string
)

const costDigit = 2

type params struct {
	Blocks    []block `json:"blocks"`
	Username  string  `json:"username"`
	IconEmoji string  `json:"icon_emoji"`
	IconURL   string  `json:"icon_url"`
	Channel   string  `json:"channel"`
}

type block struct {
	Type   string `json:"type"`
	Fields []text `json:"fields,omitempty"`
	Text   *text  `json:"text,omitempty"`
}

type text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func getEnvVal() error {
	u, ok := os.LookupEnv("SLACK_WEBHOOK_URL")
	if !ok {
		return errors.New("env SLACK_CHANNEL is not found")
	}

	slackURL = u

	c, ok := os.LookupEnv("SLACK_CHANNEL")
	if !ok {
		return errors.New("env SLACK_CHANNEL is not found")
	}

	slackChannel = c

	return nil
}

func getDateInterval() *types.DateInterval {
	const shortForm = "2006-01-02"

	t := time.Now()

	// explore cost of previous month at the beginning of month
	if t.Day() == 1 {
		return &types.DateInterval{
			Start: aws.String(t.AddDate(0, -1, 0).Format(shortForm)),
			End:   aws.String(t.AddDate(0, 0, -1).Format(shortForm)),
		}
	}

	return &types.DateInterval{
		Start: aws.String(time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC).Format(shortForm)),
		End:   aws.String(t.Format(shortForm)),
	}
}

func (c *client) getCost(dateInterval *types.DateInterval) (*costexplorer.GetCostAndUsageOutput, error) {
	getCostAndUsageInput := &costexplorer.GetCostAndUsageInput{
		Granularity: types.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		TimePeriod:  dateInterval,
		GroupBy:     []types.GroupDefinition{{Key: aws.String("SERVICE"), Type: "DIMENSION"}},
	}

	result, err := c.GetCostAndUsage(context.TODO(), getCostAndUsageInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cost and usage")
	}

	return result, nil
}

func getIndividualCost(v types.Group) (*big.Float, error) {
	c, ok := new(big.Float).SetString(*v.Metrics["UnblendedCost"].Amount)
	if !ok {
		return nil, errors.New("failed to parse big float")
	}

	return c, nil
}

func buildDetailsBlockClosure() func(keys []string, individualCost *big.Float, detailsBlocks *[]block, detailsFields *[]text) {
	var i int

	return func(keys []string, individualCost *big.Float, detailsBlocks *[]block, detailsFields *[]text) {
		if individualCost.Text('f', costDigit) == "0.00" {
			return
		}

		*detailsFields = append(*detailsFields, text{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*%s:*\n%s $", keys, individualCost.Text('f', costDigit)),
		})

		if i%2 != 0 {
			*detailsBlocks = append(*detailsBlocks, block{
				Type:   "section",
				Fields: *detailsFields,
			})

			*detailsFields = nil
		}

		i++
	}
}

func buildMainBlocks(cost *costexplorer.GetCostAndUsageOutput, total *big.Float, detailsBlocks []block) []block {
	var blocks []block

	header := block{
		Type: "header",
		Text: &text{
			Type: "plain_text",
			Text: fmt.Sprintf("Monthly AWS Cost (%s ~ %s)", *cost.ResultsByTime[0].TimePeriod.Start, *cost.ResultsByTime[0].TimePeriod.End),
		},
	}

	totalCost := block{
		Type: "section",
		Fields: []text{
			{
				Type: "mrkdwn",
				Text: fmt.Sprintf("*Total Cost:*\n%s $", total.Text('f', costDigit)),
			},
		},
	}

	divider := block{
		Type: "divider",
	}

	blocks = append(blocks, header, totalCost, divider)
	blocks = append(blocks, detailsBlocks...)

	return blocks
}

func buildResultStatement(cost *costexplorer.GetCostAndUsageOutput) (string, error) {
	var (
		detailsFields []text
		detailsBlocks []block
	)

	total := new(big.Float)

	buildDetailsBlock := buildDetailsBlockClosure()

	for _, v := range cost.ResultsByTime[0].Groups {
		individualCost, err := getIndividualCost(v)
		if err != nil {
			return "", err
		}

		total = new(big.Float).Add(total, individualCost)

		buildDetailsBlock(v.Keys, individualCost, &detailsBlocks, &detailsFields)
	}

	blocks := buildMainBlocks(cost, total, detailsBlocks)

	p := params{
		Blocks:   blocks,
		Username: "Cost Explorer",
		Channel:  slackChannel,
	}

	params, err := json.Marshal(p)
	if err != nil {
		return "", errors.Wrap(err, "failed to json marshal")
	}

	return string(params), nil
}

func (c *client) handler() error {
	dateInterval := getDateInterval()

	cost, err := c.getCost(dateInterval)
	if err != nil {
		return err
	}

	result, err := buildResultStatement(cost)
	if err != nil {
		return err
	}

	resp, err := http.PostForm(
		slackURL,
		url.Values{"payload": {result}},
	)
	if err != nil {
		return errors.Wrap(err, "failed to send Slack message")
	}

	defer resp.Body.Close()

	return nil
}

func main() {
	if err := getEnvVal(); err != nil {
		log.Fatal(err)
	}

	c, err := newClient()
	if err != nil {
		log.Fatal(err)
	}

	lambda.Start(c.handler)
}
