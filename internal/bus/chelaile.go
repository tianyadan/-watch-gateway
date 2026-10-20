package bus

import (
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const apiURL = "https://web.chelaile.net.cn/api/bus/line!encryptedBusDetail.action"

// Client
// 专门负责和车来了 API 通信。
type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {

	return &Client{
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// QueryRealtime
// 查询公交实时数据。
func (c *Client) QueryRealtime(
	ctx context.Context,
	cfg RouteConfig,
) (BusData, error) {

	params := buildParams(cfg)

	// 注意：
	//
	// 必须先用当前 params 计算签名，
	// 然后再把 cryptoSign 放进去。
	sign := cryptoSign(params)

	params = append(
		params,
		param{
			Key:   "cryptoSign",
			Value: sign,
		},
	)

	requestURL := buildURL(params)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		requestURL,
		nil,
	)

	if err != nil {
		return BusData{}, fmt.Errorf(
			"create upstream request failed: %w",
			err,
		)
	}

	setRequestHeaders(req)

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return BusData{}, fmt.Errorf(
			"connect chelaile failed: %w",
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 ||
		resp.StatusCode >= 300 {

		return BusData{}, fmt.Errorf(
			"chelaile returned HTTP %d",
			resp.StatusCode,
		)
	}

	body, err := decodeHTTPBody(resp)

	if err != nil {
		return BusData{}, err
	}

	return parseRealtimeResponse(body)
}

// buildParams
//
// 这里的参数顺序必须和你的 Python 代码完全一致。
func buildParams(cfg RouteConfig) []param {

	targetOrder := strconv.Itoa(
		cfg.TargetOrder,
	)

	return []param{

		{Key: "s", Value: "h5"},

		{Key: "wxs", Value: "wx_app"},

		{Key: "sign", Value: "1"},

		{Key: "h5RealData", Value: "1"},

		{Key: "v", Value: "3.11.28"},

		{Key: "src", Value: "weixinapp_cx"},

		{Key: "ctm_mp", Value: "mp_wx"},

		{Key: "vc", Value: "2"},

		{Key: "favoriteGray", Value: "1"},

		{Key: "gpstype", Value: "wgs"},

		{Key: "geo_type", Value: "wgs"},

		{Key: "scene", Value: "1256"},

		{Key: "cshow", Value: "busDetail"},

		{Key: "specail", Value: "0"},

		{
			Key:   "specialType",
			Value: "undefined",
		},

		{
			Key:   "cityId",
			Value: cfg.CityID,
		},

		{
			Key:   "localCityId",
			Value: cfg.CityID,
		},

		{
			Key:   "lineId",
			Value: cfg.LineID,
		},

		{
			Key:   "targetOrder",
			Value: targetOrder,
		},

		{
			Key:   "specialTargetOrder",
			Value: targetOrder,
		},

		{
			Key:   "stationId",
			Value: cfg.StationID,
		},

		{
			Key:   "lat",
			Value: cfg.StationLat,
		},

		{
			Key:   "lng",
			Value: cfg.StationLng,
		},

		{
			Key:   "geo_lat",
			Value: cfg.StationLat,
		},

		{
			Key:   "geo_lng",
			Value: cfg.StationLng,
		},

		{Key: "userId", Value: ""},

		{Key: "h5Id", Value: ""},

		{Key: "unionId", Value: ""},

		{Key: "accountId", Value: ""},

		{Key: "secret", Value: ""},
	}
}

func buildURL(params []param) string {

	parts := make(
		[]string,
		0,
		len(params),
	)

	for _, item := range params {

		parts = append(
			parts,

			url.QueryEscape(item.Key)+
				"="+
				url.QueryEscape(item.Value),
		)
	}

	return apiURL +
		"?" +
		strings.Join(parts, "&")
}

func setRequestHeaders(req *http.Request) {

	req.Header.Set(
		"Connection",
		"keep-alive",
	)

	req.Header.Set(
		"User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) "+
			"AppleWebKit/537.36 (KHTML, like Gecko) "+
			"Chrome/132.0.0.0 Safari/537.36 "+
			"MicroMessenger/7.0.20.1781(0x6700143B) "+
			"NetType/WIFI MiniProgramEnv/Windows "+
			"WindowsWechat/WMPF "+
			"WindowsWechat(0x63090a13) "+
			"UnifiedPCWindowsWechat(0xf254160a) "+
			"XWEB/18055",
	)

	req.Header.Set(
		"xweb_xhr",
		"1",
	)

	req.Header.Set(
		"Content-Type",
		"text",
	)

	req.Header.Set(
		"Accept",
		"*/*",
	)

	req.Header.Set(
		"Sec-Fetch-Site",
		"cross-site",
	)

	req.Header.Set(
		"Sec-Fetch-Mode",
		"cors",
	)

	req.Header.Set(
		"Sec-Fetch-Dest",
		"empty",
	)

	req.Header.Set(
		"Referer",
		"https://servicewechat.com/wx71d589ea01ce3321/814/page-frame.html",
	)

	req.Header.Set(
		"Accept-Encoding",
		"gzip, deflate",
	)

	req.Header.Set(
		"Accept-Language",
		"zh-CN,zh;q=0.9",
	)
}

// decodeHTTPBody
//
// 对应 Python 的:
//
// gzip.decompress
// zlib.decompress
func decodeHTTPBody(
	resp *http.Response,
) ([]byte, error) {

	var reader io.Reader = resp.Body

	encoding := strings.ToLower(
		strings.TrimSpace(
			resp.Header.Get(
				"Content-Encoding",
			),
		),
	)

	switch encoding {

	case "gzip":

		gzipReader, err := gzip.NewReader(
			resp.Body,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"create gzip reader failed: %w",
				err,
			)
		}

		defer gzipReader.Close()

		reader = gzipReader

	case "deflate":

		zlibReader, err := zlib.NewReader(
			resp.Body,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"create deflate reader failed: %w",
				err,
			)
		}

		defer zlibReader.Close()

		reader = zlibReader
	}

	body, err := io.ReadAll(reader)

	if err != nil {
		return nil, fmt.Errorf(
			"read upstream response failed: %w",
			err,
		)
	}

	return body, nil
}

// parseRealtimeResponse
//
// 上游大概结构：
//
//	{
//	    "jsonr": {
//	        "data": {
//	            "encryptResult": "..."
//	        }
//	    }
//	}
func parseRealtimeResponse(
	raw []byte,
) (BusData, error) {

	text := string(raw)

	start := strings.IndexByte(
		text,
		'{',
	)

	end := strings.LastIndexByte(
		text,
		'}',
	)

	if start < 0 ||
		end < start {

		return BusData{}, fmt.Errorf(
			"upstream response is not valid JSON",
		)
	}

	jsonBytes := raw[start : end+1]

	// 先解析外层 envelope。
	var envelope struct {
		JSONR struct {
			Data json.RawMessage `json:"data"`
		} `json:"jsonr"`
	}

	if err := json.Unmarshal(
		jsonBytes,
		&envelope,
	); err != nil {

		return BusData{}, fmt.Errorf(
			"decode upstream envelope failed: %w",
			err,
		)
	}

	if len(envelope.JSONR.Data) == 0 ||
		string(envelope.JSONR.Data) == "null" {

		return BusData{}, fmt.Errorf(
			"chelaile response missing jsonr.data",
		)
	}

	// 检查是不是加密数据。
	var probe struct {
		EncryptResult string `json:"encryptResult"`
	}

	if err := json.Unmarshal(
		envelope.JSONR.Data,
		&probe,
	); err != nil {

		return BusData{}, fmt.Errorf(
			"decode jsonr.data failed: %w",
			err,
		)
	}

	payload := envelope.JSONR.Data

	// 如果存在 encryptResult，
	// AES 解密。
	if probe.EncryptResult != "" {

		decrypted, err := decryptPayload(
			probe.EncryptResult,
		)

		if err != nil {
			return BusData{}, fmt.Errorf(
				"decrypt realtime data failed: %w",
				err,
			)
		}

		payload = decrypted
	}

	var data BusData

	if err := json.Unmarshal(
		payload,
		&data,
	); err != nil {

		return BusData{}, fmt.Errorf(
			"decode realtime data failed: %w",
			err,
		)
	}

	return data, nil
}
