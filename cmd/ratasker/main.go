package main

import (
	"context"
	"log"
	"ratasker/external/nostr"
	"ratasker/internal/database"
	"ratasker/internal/routes"
	"ratasker/internal/utils"

	w "github.com/elnosh/gonuts/wallet"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var (
	DOCKER_ENV = "DOCKER"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		c.Next()
	}
}

func main() {
	ctx := context.Background()

	sqlite, err := database.DatabaseSetup(ctx, "migrations")
	defer sqlite.Db.Close()

	if err != nil {
		log.Panicf(`database.DatabaseSetup(ctx, "migrations"). %w`, err)
	}

	r := gin.Default()

	homeDir, err := utils.GetRastaskerHomeDirectory()
	if err != nil {
		log.Panicf(`utils.GetRastaskerHomeDirectory(). %w`, err)
	}

	pathToData := homeDir + "/" + "data"

	err = utils.MakeSureFilePathExists(pathToData, "")
	if err != nil {
		log.Panicf(`utils.MakeSureFilePathExists(pathToData, ""). %w`, err)
	}

	pathToCashu := homeDir + "/" + "cashu"

	err = utils.MakeSureFilePathExists(pathToCashu, "")
	if err != nil {
		log.Panicf(`utils.MakeSureFilePathExists(pathToData, ""). %w`, err)
	}

	// Setup wallet
	config := w.Config{
		WalletPath:     pathToCashu,
		CurrentMintURL: "https://mutinynet.nutmix.cash",
	}

	wallet, err := w.LoadWallet(config)
	if err != nil {
		log.Panicf(`w.LoadWallet(config). %wa`, err)
	}
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true, // Allow all origins
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		// AllowHeaders: []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Cashu", "X-Content-Length"},
		AllowHeaders:     []string{"Authorization", "*"},
		ExposeHeaders:    []string{"Content-Length", "*"},
		AllowCredentials: true,
	}))

	routes.RootRoutes(r, wallet, sqlite)

	routes.UploadRoutes(r, wallet, sqlite, pathToData)

	log.Println("ratasker started in port 8070")
	r.Run("0.0.0.0:8070")
}

func NostrAutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHerader := c.GetHeader("Authorization")

		event, err := nostr.ParseNostrHeader(authHerader)
		if err != nil {
			c.JSON(401, nostr.NotifMessage{Message: "Missing auth event"})
			return
		}

		err = nostr.ValidateAuthEvent(event)
		if err != nil {
			c.JSON(401, nostr.NotifMessage{Message: "Invalid nostr event"})
			return
		}

		c.Set(utils.NOSTRAUTH, event)
		c.Next()
	}
}
