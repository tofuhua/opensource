package backpack

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"opensource/lib/loop"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// /api/v1/capital
type Service struct {
	Ws        string
	Key       string
	Secret    ed25519.PrivateKey
	mu        sync.Mutex
	balances  map[string]Asset
	AskPrice  string `json:"a"`
	AskAmount string `json:"A"`
	BidPrice  string `json:"b"`
	BidAmount string `json:"B"`
	Side      OrderSide
	LastTime  int64
	buyTimes  int64
	sellTimes int64
	buyValue  decimal.Decimal
	sellValue decimal.Decimal
}

const (
	balanceQuery = "https://api.backpack.exchange/api/v1/capital"
	executeOrder = "https://api.backpack.exchange/api/v1/order"
)

type OrderSide string

const (
	SideBid OrderSide = "Bid"
	SideAsk OrderSide = "Ask"
)

func New() *Service {
	srv := &Service{
		Ws:        "wss://ws.backpack.exchange/stream",
		Side:      SideAsk,
		buyValue:  decimal.NewFromFloat(0),
		sellValue: decimal.NewFromFloat(0),
	}
	key := viper.GetString("api.key")
	srv.Key = key

	secret := viper.GetString("api.secret")
	privateKey, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		panic(err)
	}
	srv.Secret = ed25519.PrivateKey(privateKey)

	return srv
}

func (srv *Service) Run() {
	log.Info("run...")
	srv.balance()

	loop.NewLoopFunc("private", srv.ListenPrivate, nil, srv.ListenPrivateStream)
	loop.NewLoopFunc("depth", srv.Listen, nil, srv.ListenStream)
	loop.Run()
	go srv.Order()
}

func (srv *Service) Stop() {
	log.Info("stop...")
	loop.Stop()
}

func (srv *Service) request(url string, method string, headers map[string]string, payload []byte) ([]byte, error) {
	var (
		req *http.Request
		err error
	)
	client := &http.Client{}
	if len(payload) > 0 {
		req, err = http.NewRequest(method, url, bytes.NewReader(payload))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (srv *Service) getHeader(sign string, params map[string]string) map[string]string {
	headers := make(map[string]string)
	headers["X-API-KEY"] = srv.Key
	headers["X-SIGNATURE"] = sign
	headers["X-TIMESTAMP"] = params["timestamp"]
	headers["X-WINDOW"] = params["window"]
	headers["Content-Type"] = "application/json; charset=utf-8"
	return headers
}
func (srv *Service) defaultParams() map[string]string {
	now := time.Now()
	timestamp := strconv.FormatInt(now.UnixMilli(), 10)
	data := make(map[string]string)
	data["timestamp"] = timestamp
	data["window"] = "5000"
	return data
}

func (srv *Service) sortSignParams(data map[string]string) string {
	values := url.Values{}
	for key, value := range data {
		values.Add(key, value)
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	// 构造最终的查询字符串
	query := values.Encode()
	return query
}

func (srv *Service) signMessage(message []byte) string {
	// 使用私钥进行签名
	signature := ed25519.Sign(srv.Secret, message)
	return base64.StdEncoding.EncodeToString(signature)
}

func (srv *Service) getAsset(symbol string) (Asset, bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	balance, ok := srv.balances[symbol]
	return balance, ok
}
