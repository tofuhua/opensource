package backpack

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"opensource/lib/constant"
	"opensource/lib/loop"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func (srv *Service) Listen(name string, options interface{}, closed chan struct{}, cf func(input interface{})) {
	bookTicker := fmt.Sprintf("%s@bookTicker", viper.GetString("exchange.symbol"))
	log.Info("Listen...")
	var subMessage = `{
		"method": "SUBSCRIBE",
		"params": ["%s"]
	}`
	subMessage = fmt.Sprintf(subMessage, bookTicker)
	u := url.URL{
		Scheme:   "wss",
		Host:     "ws.backpack.exchange",
		Path:     "/stream",
		RawQuery: "",
	}

	srv.sub(name, u, subMessage, closed, cf)

}

type OrderBookTicker struct {
	Data struct {
		Ee        string `json:"e"`
		EE        int64  `json:"E"`
		S         string `json:"s"`
		AskPrice  string `json:"a"`
		AskAmount string `json:"A"`
		BidPrice  string `json:"b"`
		BidAmount string `json:"B"`
		U         int    `json:"u"`
	} `json:"data"`
	Stream string `json:"stream"`
}

func (srv *Service) ListenStream(input interface{}) {
	data := input.([]byte)
	var order OrderBookTicker
	if err := json.Unmarshal(data, &order); err != nil {
		log.Error(err.Error())
		return
	}
	srv.mu.Lock()
	srv.AskPrice = order.Data.AskPrice
	srv.AskAmount = order.Data.AskAmount
	srv.BidPrice = order.Data.BidPrice
	srv.BidAmount = order.Data.BidAmount
	srv.mu.Unlock()
	if time.Now().UnixMilli()-srv.LastTime >= 1000*60 {
		srv.LastTime = time.Now().UnixMilli()
		log.WithFields(log.Fields{
			"卖单价格": order.Data.AskPrice,
			"卖单数量": order.Data.AskAmount,
			"买单价格": order.Data.BidPrice,
			"买单数量": order.Data.BidAmount,
		}).Info("最新")
	}

}

func (srv *Service) ListenPrivate(name string, options interface{}, closed chan struct{}, cf func(input interface{})) {
	data := srv.defaultParams()
	data["instruction"] = "accountQuery"
	signMsg := srv.sortSignParams(data)
	sign := srv.signMessage([]byte(signMsg))
	var subMessage = `
	{
		"method": "SUBSCRIBE",
		"params": ["account.orderUpdate", "%s", "%s", "%s", "%s"]
	}
`
	subMessage = fmt.Sprintf(subMessage, srv.Key, sign, data["timestamp"], data["window"])
	u := url.URL{
		Scheme:   "wss",
		Host:     "ws.backpack.exchange",
		Path:     "/stream",
		RawQuery: "",
	}
	srv.sub(name, u, subMessage, closed, cf)
}

type PrivateStream struct {
	Data struct {
		Event string `json:"e"`
		EE    int64  `json:"E"`
		Ss    string `json:"s"`
		Side  string `json:"S"`
		O     string `json:"o"`
		F     string `json:"f"`
		Q     string `json:"q"`
		P     string `json:"p"`
		X     string `json:"X"`
		I     string `json:"i"`
		Ll    string `json:"l"`
		Zz    string `json:"z"`
		ZZ    string `json:"Z"`
		LL    string `json:"L"`
		M     bool   `json:"m"`
		Nn    string `json:"n"`
		NN    string `json:"N"`
		V     string `json:"V"`
	} `json:"data"`
	Stream string `json:"stream"`
}

func (srv *Service) ListenPrivateStream(input interface{}) {
	// data := input.([]byte)

	// var payload PrivateStream
	// if err := json.Unmarshal(data, &payload); err != nil {
	// 	log.WithFields(log.Fields{
	// 		"data": string(data),
	// 	}).Error(err.Error())
	// 	return
	// }
	// log.WithFields(log.Fields{
	// 	"data": string(data),
	// }).Info("订单更新")
}

func (srv *Service) sub(name string, u url.URL, subMsg string, closed chan struct{}, cf func(interface{})) {
	log.Infof("connecting to %s", u.String())
	c, resp, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		body, _ := io.ReadAll(resp.Body)
		log.Errorf("dial err:%s,name:%s,statusCode:%s,body:%s", err, name, resp.Status, string(body))
		loop.SetState(name, constant.StatusInactive)
		return
	}
	defer c.Close()
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Errorf("read 001:%v", err)
				return
			}
			cf(message)
		}
	}()
	log.Infof("send sub message:%s", name)
	err = c.WriteMessage(websocket.TextMessage, []byte(subMsg))
	if err != nil {
		log.Errorf("write 002:%v", err)
		loop.SetState(name, constant.StatusInactive)
		return
	}
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			log.Info("done", name)
			loop.SetState(name, constant.StatusInactive)
			return
		case <-ticker.C:

		case <-closed:
			log.Info("closed", name)
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Errorf("write close:%v", err)
				return
			}
			select {
			case <-done:
				log.Info("case done", name)
			case <-time.After(time.Second):
			}
			return
		}
	}
}
