package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stateshape/augur-analyzer/pkg/augur"
	"github.com/stateshape/augur-analyzer/pkg/env"
	"github.com/stateshape/augur-analyzer/pkg/gcloud"
	"github.com/stateshape/augur-analyzer/pkg/markets"
	"github.com/stateshape/augur-analyzer/pkg/pricing"
	"github.com/stateshape/augur-analyzer/pkg/web3"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func environment() {
	viper.SetDefault(env.EthereumHostWS, "")
	viper.SetDefault(env.EthereumHostHTTP, "")
	viper.SetDefault(env.CoinbaseAPIKey, "")
	viper.SetDefault(env.CoinbaseAPISecret, "")
	viper.SetDefault(env.AugurGRPCHost, "localhost")
	viper.SetDefault(env.AugurGRPCPort, "50051")
	viper.SetDefault(env.AugurRootUniverse, "")
	viper.SetDefault(env.HTTPServerPort, "49990")
	viper.SetDefault(env.HTTPServerNetworkInterface, "localhost")
	viper.SetDefault(env.GoogleApplicationCredentials, "")
	viper.SetDefault(env.GCloudProjectID, "predictions-global-prod")
	viper.SetDefault(env.GCloudStorageBucket, "predictions-global-prod")
	viper.SetDefault(env.GCloudStorageMarketsObjectName, "markets.pb")
	viper.AutomaticEnv()

	required := []string{
		env.EthereumHostHTTP,
		env.CoinbaseAPIKey,
		env.CoinbaseAPISecret,
	}
	for _, envvar := range required {
		if viper.GetString(envvar) == "" {
			logrus.Panicf("Environment variable `%s` is required.", envvar)
		}
	}
}

func main() {
	environment()

	// Web3 API
	web3API, err := web3.NewClient(web3.EthereumHosts{
		viper.GetString(env.EthereumHostWS),
		viper.GetString(env.EthereumHostHTTP),
	})
	if err != nil {
		logrus.WithError(err).Panicf("Failed to create a web3 client")
	}

	// Digital asset pricing API
	pricingAPI := pricing.NewCoinbasePricingClient(
		viper.GetString(env.CoinbaseAPIKey),
		viper.GetString(env.CoinbaseAPISecret),
	)
	if err != nil {
		logrus.WithError(err).Panicf("Failed to create a pricing client")
	}
	logrus.Printf("Pricing API Client: %#v", pricingAPI)

	// GCloud API
	gcloudStorageAPI, err := gcloud.NewStorageClient()
	if err != nil {
		logrus.WithError(err).Panicf("Failed to create a GCloud Storge client")
	}

	// augur-node GRPC API
	grpcHost := fmt.Sprintf("%s:%s", viper.GetString(env.AugurGRPCHost), viper.GetString(env.AugurGRPCPort))
	augurAPIConn, err := grpc.Dial(grpcHost, grpc.WithInsecure())
	if err != nil {
		logrus.WithError(err).Panicf("Failed to create client for augur-grpc")
	}
	augurAPI := augur.NewMarketsApiClient(augurAPIConn)

	watcher := markets.NewWatcher(pricingAPI, web3API, augurAPI, gcloudStorageAPI)
	go watcher.Watch()

	// Start HTTP server
	r := gin.Default()
	r.Run(fmt.Sprintf("%s:%s", viper.GetString(env.HTTPServerNetworkInterface), viper.GetString(env.HTTPServerPort)))

	// Wait for OS termination signal
	end := make(chan os.Signal)
	signal.Notify(end, syscall.SIGTERM)
	<-end
	logrus.Infof("SIGTERM signal received, shutting down at unix time: %d", time.Now().Unix())
}
