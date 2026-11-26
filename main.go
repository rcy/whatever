package main

import (
	"log"
	"os"
	_ "time/tzdata"

	"github.com/alecthomas/kong"
	"github.com/joho/godotenv"
	"github.com/rcy/whatever/app"
	"github.com/rcy/whatever/cli"
)

func main() {
	_ = godotenv.Load()
	filename, ok := os.LookupEnv("EVOKE_FILE")
	if !ok {
		log.Fatal("EVOKE_FILE not set")
	}
	a, err := app.New(filename, app.Config{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	})
	if err != nil {
		log.Fatal(err)
	}

	kctx := kong.Parse(&cli.CLI)
	err = kctx.Run(a)
	kctx.FatalIfErrorf(err)
}
