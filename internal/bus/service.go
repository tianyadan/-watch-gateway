package bus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// FlexibleNumber
//
// 车来了有些字段可能返回：
//
// 123
//
// 也可能返回：
//
// "123"
//
// 所以不能单纯使用 int。
type FlexibleNumber struct {
	Value float64

	Valid bool
}

func (n *FlexibleNumber) UnmarshalJSON(
	data []byte,
) error {

	data = bytes.TrimSpace(data)

	if bytes.Equal(
		data,
		[]byte("null"),
	) || len(data) == 0 {

		n.Valid = false

		return nil
	}

	var value float64

	// 字符串数字
	if data[0] == '"' {

		var text string

		if err := json.Unmarshal(
			data,
			&text,
		); err != nil {

			return err
		}

		if text == "" {

			n.Valid = false

			return nil
		}

		parsed, err := strconv.ParseFloat(
			text,
			64,
		)

		if err != nil {
			return err
		}

		value = parsed

	} else {

		// 普通 JSON Number
		if err := json.Unmarshal(
			data,
			&value,
		); err != nil {

			return err
		}
	}

	n.Value = value

	n.Valid = true

	return nil
}

// BusData
// AES 解密之后我们真正关心的数据。
type BusData struct {
	RealData bool `json:"realData"`

	Buses []Bus `json:"buses"`
}

type Bus struct {
	Order FlexibleNumber `json:"order"`

	Licence string `json:"licence"`

	BusID string `json:"busId"`

	DistanceToWaitStn FlexibleNumber `json:"distanceToWaitStn"`

	Travels []Travel `json:"travels"`
}

type Travel struct {
	TravelTime FlexibleNumber `json:"travelTime"`

	RecommTip string `json:"recommTip"`

	ArrivalTime FlexibleNumber `json:"arrivalTime"`
}

// Service
//
// 业务层。
type Service struct {
	client *Client
}

func NewService(
	client *Client,
) *Service {

	return &Service{
		client: client,
	}
}

// RouteData
//
// 最终返回给调用方的数据。
//
// 这里已经彻底隐藏了
// lineId / stationId 等第三方字段。
type RouteData struct {
	Route string `json:"route"`

	City string `json:"city"`

	Scene string `json:"scene"`

	Line string `json:"line"`

	From string `json:"from"`

	To string `json:"to"`

	RealData bool `json:"realData"`

	QueryTime string `json:"queryTime"`

	Buses []BusResult `json:"buses"`
}

type BusResult struct {
	Plate string `json:"plate"`

	DistanceMeters int `json:"distanceMeters"`

	DistanceText string `json:"distanceText"`

	RemainingSeconds int `json:"remainingSeconds"`

	RemainingText string `json:"remainingText"`

	ArrivalTime string `json:"arrivalTime"`
}

type candidate struct {
	Bus Bus

	Travel Travel
}

func (s *Service) QueryRoute(
	ctx context.Context,
	cfg RouteConfig,
) (RouteData, error) {

	data, err := s.client.QueryRealtime(
		ctx,
		cfg,
	)

	if err != nil {
		return RouteData{}, err
	}

	items := nearestBuses(
		data,
		cfg.TargetOrder,
	)

	results := make(
		[]BusResult,
		0,
		len(items),
	)

	for _, item := range items {

		seconds := int(
			item.Travel.TravelTime.Value,
		)

		// 默认 -1 代表未知。
		distance := -1

		if item.Bus.DistanceToWaitStn.Valid &&
			item.Bus.DistanceToWaitStn.Value >= 0 {

			// 四舍五入
			distance = int(
				item.Bus.DistanceToWaitStn.Value +
					0.5,
			)
		}

		plate := item.Bus.Licence

		if plate == "" {
			plate = item.Bus.BusID
		}

		if plate == "" {
			plate = "未知"
		}

		arrival := item.Travel.RecommTip

		// 如果 recommTip 没有返回，
		// 尝试使用 arrivalTime。
		if arrival == "" &&
			item.Travel.ArrivalTime.Valid &&
			item.Travel.ArrivalTime.Value > 0 {

			arrival = time.UnixMilli(
				int64(
					item.Travel.ArrivalTime.Value,
				),
			).
				In(
					shanghaiLocation(),
				).
				Format("15:04")
		}

		if arrival == "" {
			arrival = "未知"
		}

		results = append(
			results,

			BusResult{
				Plate: plate,

				DistanceMeters: distance,

				DistanceText: formatDistance(
					distance,
				),

				RemainingSeconds: seconds,

				RemainingText: formatRemaining(
					seconds,
				),

				ArrivalTime: arrival,
			},
		)
	}

	return RouteData{
		Route:    cfg.Name(),
		City:     cfg.City,
		Scene:    cfg.Scene,
		Line:     cfg.Line,
		From:     cfg.From,
		To:       cfg.To,
		RealData: data.RealData,
		QueryTime: time.Now().
			In(shanghaiLocation()).
			Format(time.RFC3339),
		Buses: results,
	}, nil
}

// nearestBuses
//
// 对应你的 Python nearest_buses。
func nearestBuses(
	data BusData,
	targetOrder int,
) []candidate {

	items := make(
		[]candidate,
		0,
		len(data.Buses),
	)

	for _, bus := range data.Buses {

		if len(bus.Travels) == 0 {
			continue
		}

		// 已经过目标站点的车不需要。
		if bus.Order.Valid &&
			int(bus.Order.Value) > targetOrder {

			continue
		}

		travel := bus.Travels[0]

		// 必须有有效的到站时间。
		if !travel.TravelTime.Valid ||
			travel.TravelTime.Value <= 0 {

			continue
		}

		items = append(
			items,

			candidate{
				Bus:    bus,
				Travel: travel,
			},
		)
	}

	// 根据到站时间排序。
	sort.Slice(
		items,

		func(i, j int) bool {

			return items[i].
				Travel.
				TravelTime.
				Value <
				items[j].
					Travel.
					TravelTime.
					Value
		},
	)

	// 最多只要两辆。
	if len(items) > 2 {
		items = items[:2]
	}

	return items
}

func formatRemaining(
	seconds int,
) string {

	if seconds < 60 {

		return fmt.Sprintf(
			"约 %d 秒",
			seconds,
		)
	}

	minutes := seconds / 60

	remain := seconds % 60

	if remain == 0 {

		return fmt.Sprintf(
			"约 %d 分钟",
			minutes,
		)
	}

	return fmt.Sprintf(
		"约 %d 分 %d 秒",
		minutes,
		remain,
	)
}

func formatDistance(
	meters int,
) string {

	if meters < 0 {
		return "未知"
	}

	return fmt.Sprintf(
		"%d 米",
		meters,
	)
}

// 服务器部署到 Linux 后，
// 系统默认时区可能是 UTC。
//
// 所以这里主动使用 Asia/Shanghai。
func shanghaiLocation() *time.Location {

	location, err := time.LoadLocation(
		"Asia/Shanghai",
	)

	if err == nil {
		return location
	}

	// 极端情况下服务器没有 timezone database，
	// 直接固定 UTC+8。
	return time.FixedZone(
		"CST",
		8*60*60,
	)
}
