package cmd

import (
	"compound/worker"
	"compound/worker/market"
	"compound/worker/priceoracle"
	"compound/worker/snapshot"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "compound job worker",
	Run: func(cmd *cobra.Command, args []string) {
		// ctx := cmd.Context()

		db := provideDatabase()
		dapp := provideDApp()
		blockWallet := provideBlockWallet()
		config := provideConfig()

		propertyStore := providePropertyStore(db)
		marketStore := provideMarketStore()

		walletService := provideWalletService()
		blockService := provideBlockService()
		priceService := providePriceService()
		marketService := provideMarketService()

		workers := []worker.IJob{
			priceoracle.New(dapp, blockWallet, config, marketStore, blockService, priceService),
			market.New(dapp, blockWallet, config, marketStore, blockService, priceService),
			snapshot.New(config, dapp, propertyStore, walletService, priceService, blockService, marketService),
		}

		for _, w := range workers {
			w.Start()
			defer w.Stop()
		}

		sig := make(chan os.Signal)
		signal.Notify(sig, os.Interrupt, os.Kill)
		<-sig
	},
}

func init() {
	rootCmd.AddCommand(workerCmd)
}
