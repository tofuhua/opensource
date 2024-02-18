package backpack

import (
	"encoding/json"
	"net/http"
	"opensource/lib/status"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

/*
{"side":"Ask","symbol":"SOL_USDC","orderType":"Limit","timeInForce":"IOC","quantity":"0.1","price":"62.46"}
*/
type Order struct {
	Side      string `json:"side"`
	Symbol    string `json:"symbol"`
	OrderType string `json:"orderType"`
	Price     string `json:"price"`
	Quantity  string `json:"quantity"`
}

type OrderResponse struct {
	CreatedAt             int64  `json:"createdAt"`
	ExecutedQuantity      string `json:"executedQuantity"`
	ExecutedQuoteQuantity string `json:"executedQuoteQuantity"`
	ID                    string `json:"id"`
	OrderType             string `json:"orderType"`
	PostOnly              bool   `json:"postOnly"`
	Price                 string `json:"price"`
	Quantity              string `json:"quantity"`
	SelfTradePrevention   string `json:"selfTradePrevention"`
	Side                  string `json:"side"`
	Status                string `json:"status"`
	Symbol                string `json:"symbol"`
	TimeInForce           string `json:"timeInForce"`
}

func (srv *Service) Order() {
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()
	status.AddWaitGroup()
	defer status.DoneWaitGroup()
	for status.IsRunning() {
		select {
		case <-ticker.C:
			srv.balance()
			if srv.Side != SideBid {
				if srv.BidPrice == "" {
					continue
				}
				srv.Side = SideBid
				srv.Buy()
			} else {
				if srv.AskPrice == "" {
					continue
				}
				srv.Side = SideAsk
				srv.Sell()
			}
		}
	}
}

func (srv *Service) Buy() {
	log.Info("buy begin")
	balance, ok := srv.getAsset(viper.GetString("exchange.buy_symbol"))
	if !ok {
		log.Error("buy token not exist")
		return
	}

	value, err := strconv.ParseFloat(balance.Available, 64)
	if err != nil {
		log.Error(err.Error())
		return
	}

	buyValue := viper.GetFloat64("exchange.amount")
	price, _ := strconv.ParseFloat(srv.AskPrice, 64)

	if value < buyValue*price {
		log.Info("buy value not enough,pls wait next times")
		return
	}

	order := Order{
		Side:      string(SideBid),
		Symbol:    viper.GetString("exchange.symbol"),
		OrderType: "Limit",
		Price:     srv.AskPrice,
		Quantity:  viper.GetString("exchange.amount"),
	}
	data := srv.defaultParams()
	data["instruction"] = "orderExecute"
	data["side"] = order.Side
	data["symbol"] = order.Symbol
	data["orderType"] = order.OrderType
	data["price"] = order.Price
	data["quantity"] = order.Quantity

	signMsg := srv.sortSignParams(data)
	sign := srv.signMessage([]byte(signMsg))
	payload, _ := json.Marshal(order)
	result, err := srv.request(executeOrder, http.MethodPost, srv.getHeader(sign, data), payload)
	if err != nil {
		log.Error("buy err:", err.Error())
		return
	}
	var resp OrderResponse
	log.Info("buy:", string(result))
	if err := json.Unmarshal(result, &resp); err != nil {
		log.Error("buy err:", err.Error())
		return
	}
	srv.buyTimes += 1
	payValue := decimal.NewFromFloat(buyValue).Mul(decimal.NewFromFloat(price))
	srv.buyValue = srv.buyValue.Add(payValue)
	log.WithFields(log.Fields{
		"buyTimes":  srv.buyTimes,
		"sellTimes": srv.sellTimes,
		"buyValue":  srv.buyValue,
		"sellValue": srv.sellValue,
	}).Info("buy success")

	srv.balance()
}

func (srv *Service) Sell() {
	log.Info("sell begin")

	balance, ok := srv.getAsset(viper.GetString("exchange.sell_symbol"))
	if !ok {
		log.Error("sell token not exist")
		return
	}

	value, err := strconv.ParseFloat(balance.Available, 64)
	if err != nil {
		log.Error(err.Error())
		return
	}

	sellValue := viper.GetFloat64("exchange.amount")
	if value < sellValue {
		log.Info("sell value not enough,pls wait next times")
		return
	}

	data := srv.defaultParams()

	order := Order{
		Side:      string(SideAsk),
		Symbol:    viper.GetString("exchange.symbol"),
		OrderType: "Limit",
		Price:     srv.BidPrice,
		Quantity:  viper.GetString("exchange.amount"),
	}
	data["instruction"] = "orderExecute"
	data["side"] = order.Side
	data["symbol"] = order.Symbol
	data["orderType"] = order.OrderType
	data["price"] = order.Price
	data["quantity"] = order.Quantity

	signMsg := srv.sortSignParams(data)
	sign := srv.signMessage([]byte(signMsg))
	payload, _ := json.Marshal(order)
	result, err := srv.request(executeOrder, http.MethodPost, srv.getHeader(sign, data), payload)
	if err != nil {
		log.Error("sell err:", err.Error())
		return
	}
	var resp OrderResponse
	log.Info("sell:", string(result))
	if err := json.Unmarshal(result, &resp); err != nil {
		log.Error("sell err:", err.Error())
		return
	}

	srv.sellTimes += 1
	srv.sellValue = srv.sellValue.Add(decimal.NewFromFloat(sellValue))
	log.WithFields(log.Fields{
		"buyTimes":  srv.buyTimes,
		"sellTimes": srv.sellTimes,
		"buyValue":  srv.buyValue,
		"sellValue": srv.sellValue,
	}).Info("sell success")
	srv.balance()
}
