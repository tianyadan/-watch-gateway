package bus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const barkAPIURL = "https://api.day.app/push"

type BarkClient struct {
	deviceKey  string
	httpClient *http.Client
}

type barkRequest struct {
	DeviceKey string `json:"device_key"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Group     string `json:"group"`
}

type barkResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func NewBarkClientFromEnv() *BarkClient {
	return &BarkClient{
		deviceKey: strings.TrimSpace("sEpDiqXTbec8aEx2VUCN2N"),
		httpClient: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

// PushBusResult
//
// 公交查询成功后，把结果推送到 Bark。
func (c *BarkClient) PushBusResult(
	ctx context.Context,
	cfg RouteConfig,
	data RouteData,
) error {

	if c == nil {
		return errors.New("bark client is nil")
	}

	if c.deviceKey == "" {
		return errors.New("BARK_KEY environment variable is empty")
	}

	title := buildBarkTitle(cfg)

	body := buildBarkBody(data)

	payload := barkRequest{
		DeviceKey: c.deviceKey,
		Title:     title,
		Body:      body,
		Group:     "bus",
	}

	requestBody, err := json.Marshal(payload)

	if err != nil {
		return fmt.Errorf(
			"marshal bark request failed: %w",
			err,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		barkAPIURL,
		bytes.NewReader(requestBody),
	)

	if err != nil {
		return fmt.Errorf(
			"create bark request failed: %w",
			err,
		)
	}

	req.Header.Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return fmt.Errorf(
			"request bark failed: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 ||
		resp.StatusCode >= 300 {

		return fmt.Errorf(
			"bark returned HTTP %d",
			resp.StatusCode,
		)
	}

	var result barkResponse

	if err := json.NewDecoder(resp.Body).Decode(
		&result,
	); err != nil {

		return fmt.Errorf(
			"decode bark response failed: %w",
			err,
		)
	}

	// Bark 成功返回 code = 200
	if result.Code != 200 {

		return fmt.Errorf(
			"bark push failed: code=%d message=%s",
			result.Code,
			result.Message,
		)
	}

	return nil
}

func buildBarkTitle(cfg RouteConfig) string {
	line := cfg.Line
	if line == "" {
		line = "公交"
	}
	scene := cfg.Scene
	if scene == "" {
		scene = cfg.From
	}
	if scene == "" {
		scene = cfg.Name()
	}
	return fmt.Sprintf("%s 路公交车 %s 查询结果", line, scene)
}

func buildBarkBody(data RouteData) string {

	if len(data.Buses) == 0 {

		return fmt.Sprintf(
			"%s → %s\n暂未查询到即将到站的公交车。",
			data.From,
			data.To,
		)
	}

	var builder strings.Builder

	builder.WriteString(
		fmt.Sprintf(
			"%s → %s\n",
			data.From,
			data.To,
		),
	)

	for index, bus := range data.Buses {

		if index > 0 {
			builder.WriteString("\n")
		}

		builder.WriteString(
			fmt.Sprintf(
				"第%d辆：%s\n",
				index+1,
				bus.RemainingText,
			),
		)

		builder.WriteString(
			fmt.Sprintf(
				"距离：%s\n",
				bus.DistanceText,
			),
		)

		builder.WriteString(
			fmt.Sprintf(
				"预计到站：%s",
				bus.ArrivalTime,
			),
		)

		if bus.Plate != "" &&
			bus.Plate != "未知" {

			builder.WriteString(
				fmt.Sprintf(
					"\n车牌：%s",
					bus.Plate,
				),
			)
		}
	}

	return builder.String()
}
