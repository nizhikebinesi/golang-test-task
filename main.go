package main

import (
	"fmt"
	"golang-test-task/database"
	"golang-test-task/facade"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-playground/validator/v10"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @Title Ad Submitting Service
// @Version 0.1
// @Description A sample server for creating and getting ads

// @Contact.name Denis Skripov
// @Contact.email nizhikebinesi@gmail.com

// @Host localhost:8888
// @BasePath /api/v0.1

func main() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ := config.Build(zap.Hooks(func(entry zapcore.Entry) error {
		if entry.Level == zapcore.DebugLevel ||
			entry.Level == zapcore.WarnLevel ||
			entry.Level == zapcore.ErrorLevel ||
			entry.Level == zapcore.PanicLevel ||
			entry.Level == zapcore.DPanicLevel {
			defer sentry.Flush(2 * time.Second)
			sentry.CaptureMessage(fmt.Sprintf("%s, Line No: %d :: %s", entry.Caller.File, entry.Caller.Line, entry.Message))
		}
		return nil
	}))
	defer func() {
		_ = logger.Sync()
	}()

	// https://github.com/shopspring/decimal/issues/21
	decimal.MarshalJSONWithoutQuotes = true

	// TODO: sync it with git tags
	apiVersion := os.Getenv("API_VERSION")

	dsn := os.Getenv("DB_DSN")

	// TODO: add zap to sentry - https://github.com/TheZeroSlave/zapsentry
	sentryDSN := os.Getenv("SENTRY_DSN")
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              sentryDSN,
		Release:          fmt.Sprintf("golang-test-task@%s", apiVersion),
		Debug:            true,
		TracesSampleRate: 1.0,
	})
	if err != nil {
		logger.Panic("sentry does not init", zap.Error(err))
	}

	v := validator.New()
	_ = v.RegisterValidation("checkURL", func(fl validator.FieldLevel) bool {
		arr, ok := fl.Field().Interface().([]string)
		if !ok {
			return false
		}
		for _, a := range arr {
			_, err := url.ParseRequestURI(a)
			if err != nil {
				return false
			}
		}
		return true
	})

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	err = db.AutoMigrate(&database.AdItem{}, &database.ImageURL{})
	if err != nil {
		logger.Panic("failed to automigrate", zap.Error(err))
	}

	client := database.NewClient(db)
	logic := facade.NewHandlerFacade(client, v, logger)

	mux := http.NewServeMux()
	endpoints := []string{"create_ad", "get_ad", "list_ads"}
	for _, endpoint := range endpoints {
		path := fmt.Sprintf("/api/v%s/%s", apiVersion, endpoint)
		if h, ok := logic.GetHandler(endpoint); ok {
			mux.HandleFunc(path, h)
		} else {
			logger.Warn("handler endpoint does not contain in logic", zap.String("endpoint", endpoint))
		}
	}
	err = http.ListenAndServe(":3000", mux)
	if err != nil {
		logger.Panic("not nil serving", zap.Error(err))
	}
}
