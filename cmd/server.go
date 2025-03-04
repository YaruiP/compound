package cmd

import (
	"compound/handler/hc"
	"compound/handler/rest"
	"fmt"
	"net/http"

	"github.com/fox-one/pkg/logger"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
)

// command for restful api server running
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run compound api server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		log := logger.FromContext(ctx)

		migrateDB()

		db := provideDatabase()
		defer db.Close()

		userStore := provideUserStore(db)
		marketStore := provideMarketStore(db)
		supplyStore := provideSupplyStore(db)
		borrowStore := provideBorrowStore(db)
		transactionStore := provideTransactionStore(db)

		blockService := provideBlockService()
		priceService := providePriceService(blockService)
		marketService := provideMarketService(marketStore, blockService)
		accountService := provideAccountService(marketStore, supplyStore, borrowStore, priceService, blockService, marketService)

		mux := chi.NewMux()
		mux.Use(middleware.Recoverer)
		mux.Use(middleware.StripSlashes)
		mux.Use(cors.AllowAll().Handler)
		mux.Use(logger.WithRequestID)
		mux.Use(middleware.Logger)
		mux.Use(middleware.NewCompressor(5).Handler)

		{
			//hc for health check
			mux.Mount("/hc", hc.Handle(rootCmd.Version))
		}

		{
			//restful api
			mux.Mount("/api/v1", rest.Handle(userStore, marketStore, supplyStore, borrowStore, transactionStore, blockService, priceService, accountService, marketService))
		}

		port, _ := cmd.Flags().GetInt("port")
		addr := fmt.Sprintf(":%d", port)

		server := &http.Server{
			Addr:    addr,
			Handler: mux,
		}

		log.Infoln("serve at", addr)
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.WithError(err).Fatal("server aborted")
		}
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)
	serverCmd.Flags().IntP("port", "p", 80, "server port")
}
