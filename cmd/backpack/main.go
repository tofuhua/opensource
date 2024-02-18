package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"opensource/internal/backpack"
	"opensource/lib/status"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

var (
	configFile string
)

func init() {
	flag.StringVar(&configFile, "c", "", "eg: -c etc")

}
func main() {
	flag.Parse()
	viper.SetConfigName("backpack")
	viper.SetConfigType("yaml")

	viper.AddConfigPath(configFile)

	// 读取配置文件，如果没有找到配置文件或出错，会产生错误
	if err := viper.ReadInConfig(); err != nil {
		log.Errorf("无法读取配置文件:%s\n", err)
		return
	}

	if viper.GetString("api.key") == "" {
		pub, priv, _ := ed25519.GenerateKey((rand.Reader))
		viper.Set("api.key", base64.StdEncoding.EncodeToString(pub))
		viper.Set("api.secret", base64.StdEncoding.EncodeToString(priv))
		if err := viper.WriteConfig(); err != nil {
			log.Error(err)
			return
		}
		log.WithFields(log.Fields{
			"api.key":    base64.StdEncoding.EncodeToString(pub),
			"api.secret": base64.StdEncoding.EncodeToString(priv),
		}).Info("配置成功 请将 api.key 添加到网站")
		return
	}

	srv := backpack.New()
	go srv.Run()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, syscall.SIGINT)
	<-sig
	status.Shutdown()
	status.WaitGroup()
	srv.Stop()

}
