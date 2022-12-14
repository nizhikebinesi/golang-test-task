package entities

import "github.com/shopspring/decimal"

type (
	// GetAdAnswer couples a status of processing and a request's result
	GetAdAnswer struct {
		Status string     `json:"status"`
		Result *APIAdItem `json:"result"`
	}

	// APIAdItem is a struct to bind data for getAd
	APIAdItem struct {
		ID           int             `json:"id"`
		Title        string          `json:"title"`
		Price        decimal.Decimal `json:"price"`
		MainImageURL *string         `json:"main_image_url"`
		Description  string          `json:"description,omitempty"`
		ImageURLs    []string        `json:"image_urls,omitempty"`
	}
)
