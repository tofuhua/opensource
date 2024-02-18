package backpack

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type Asset struct {
	Available string `json:"available"`
	Locked    string `json:"locked"`
	Staked    string `json:"staked"`
}

func (srv *Service) balance() {
	log.Info("balanceQuery")
	data := srv.defaultParams()
	data["instruction"] = "balanceQuery"
	signMsg := srv.sortSignParams(data)
	sign := srv.signMessage([]byte(signMsg))
	result, err := srv.request(balanceQuery, http.MethodGet, srv.getHeader(sign, data), []byte{})
	if err != nil {
		log.Error("balanceQuery:", err.Error())
		return
	}
	//log.Info("balanceQuery:", string(result))
	var balances map[string]Asset
	if err := json.Unmarshal(result, &balances); err != nil {
		log.Error("balanceQuery:", err.Error())
		return
	}
	for symbol, v := range balances {
		log.WithFields(log.Fields{
			"available": v.Available,
			"locked":    v.Locked,
			"staked":    v.Staked,
		}).Info(symbol)
	}
	srv.balances = balances
}
