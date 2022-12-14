package facade

import (
	"fmt"
	"golang-test-task/internal/cache"
	"golang-test-task/internal/database"
	"golang-test-task/internal/entities"
	"io"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/mailru/easyjson"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type (
	// HandlerFacade is helper struct for getting handlers and hiding inner logic of an app
	HandlerFacade struct {
		dbClient     *database.Client
		validator    *validator.Validate
		logger       *zap.Logger
		handlers     map[string]handler
		singleflight *singleflight.Group
		cache        *cache.RedisClient
	}
	handler   func(w http.ResponseWriter, r *http.Request)
	handlerBs func(w http.ResponseWriter, r *http.Request, bs []byte)
)

// NewHandlerFacade is constructor for HandlerFacade
func NewHandlerFacade(cache *cache.RedisClient, dbClient *database.Client, validator *validator.Validate, logger *zap.Logger) *HandlerFacade {
	facade := HandlerFacade{dbClient: dbClient, validator: validator, logger: logger}
	facade.handlers = make(map[string]handler)
	facade.handlers["create_ad"] = facade.readAllWrap(facade.createAd)
	facade.handlers["get_ad"] = facade.getAd
	facade.handlers["list_ads"] = facade.listAds
	facade.singleflight = &singleflight.Group{}
	facade.cache = cache
	return &facade
}

// GetHandler gives a handler based on endpoint
func (hf *HandlerFacade) GetHandler(s string) (func(w http.ResponseWriter, r *http.Request), bool) {
	h, ok := hf.handlers[s]
	return h, ok
}

func (hf *HandlerFacade) readAllWrap(h handlerBs) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		bs, err := io.ReadAll(r.Body)
		if err != nil {
			hf.logger.Error("error during ReadAll")
			result := entities.GetAdAnswer{Status: "error"}
			bs, _ = easyjson.Marshal(result)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(bs)
			return
		}
		h(w, r, bs)
	}
}

// listAds is a function to get list of ads
//
// sorting by price/date_created; asc/desc order
// TODO: add pagination size to params
// @Accept  json
// @Description An endpoint to get item list for pagination
// @Failure 422 {object} entities.ListAdsAnswer{}
// @Failure 500 {object} entities.ListAdsAnswer{}
// @Success 200 {object} entities.ListAdsAnswer{}
// @Router /list_ads [get]
func (hf *HandlerFacade) listAds(w http.ResponseWriter, r *http.Request) {
	var bs []byte
	result := entities.ListAdsAnswer{Status: "error"}

	params := r.URL.Query()
	offsetStrings := params["offset"]

	var offset int
	if len(offsetStrings) > 0 {
		offsetInt, err := strconv.Atoi(offsetStrings[0])
		if err != nil {
			hf.logger.Error(
				"cannot cast offset to int in listAds",
				zap.Error(err), zap.Strings("offsetStrings", offsetStrings),
				zap.String("offsetStrings[0]", offsetStrings[0]))
			w.WriteHeader(http.StatusUnprocessableEntity)
			bs, _ = easyjson.Marshal(result)
			_, _ = w.Write(bs)
			return
		}
		if offsetInt < 0 {
			hf.logger.Error(
				"offset is negative integer in listAds",
				zap.Error(err), zap.Strings("offsetStrings", offsetStrings),
				zap.String("offsetStrings[0]", offsetStrings[0]))
			w.WriteHeader(http.StatusUnprocessableEntity)
			bs, _ = easyjson.Marshal(result)
			_, _ = w.Write(bs)
			return
		}
		offset = offsetInt
	}

	byStrings := params["by"]
	var by string
	if len(byStrings) == 0 {
		by = entities.ByCreatedAt
	} else {
		by = byStrings[0]
	}
	if by != entities.ByCreatedAt && by != entities.ByPrice {
		hf.logger.Error("incorrect by param in listAds",
			zap.Strings("byStrings", byStrings))
		w.WriteHeader(http.StatusUnprocessableEntity)
		bs, _ = easyjson.Marshal(result)
		_, _ = w.Write(bs)
		return
	}

	ascStrings := params["asc"]
	var asc = true
	if len(ascStrings) > 0 {
		ascBool, err := strconv.ParseBool(ascStrings[0])
		if err != nil {
			hf.logger.Error(
				"cannot cast asc to bool in listAds",
				zap.Error(err), zap.Strings("ascStrings", ascStrings),
				zap.String("ascStrings[0]", ascStrings[0]))
			w.WriteHeader(http.StatusUnprocessableEntity)
			bs, _ = easyjson.Marshal(result)
			_, _ = w.Write(bs)
			return
		}
		asc = ascBool
	}

	itms, err := hf.dbClient.ListAds(offset, entities.PaginationSize, by, asc)
	if err != nil {
		hf.logger.Error("error during dbClient.listAds in listAds", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		bs, _ = easyjson.Marshal(result)
		_, _ = w.Write(bs)
		return
	}

	readyItems := make([]entities.APIAdListItem, len(itms))
	for i, v := range itms {
		readyItems[i] = v.CreateMap()
	}

	result.Result = readyItems
	bs, _ = easyjson.Marshal(result)
	_, _ = w.Write(bs)
}

// getAd is a function to get concreate ad
//
// required fields: title, price, main_photo_url
// additional: by parameter `fields`(description, photo_urls)
// @Accept  json
// @Description An endpoint to get item by id
// @Failure 422 {object} entities.GetAdAnswer{}
// @Failure 500 {object} entities.GetAdAnswer{}
// @Success 200 {object} entities.GetAdAnswer{}
// @Router /get_ad [get]
func (hf *HandlerFacade) getAd(w http.ResponseWriter, r *http.Request) {
	result := entities.GetAdAnswer{Status: "error"}
	vars := mux.Vars(r)
	idx := vars["id"]
	var bs []byte
	id, _ := strconv.Atoi(idx)

	queryFields := r.URL.Query()["fields"]
	var fieldsMap = make(map[string]struct{})
	for _, field := range queryFields {
		if field != "description" && field != "image_urls" {
			hf.logger.Error("field is not acceptable", zap.String("field", field))
			w.WriteHeader(http.StatusUnprocessableEntity)
			bs, _ = easyjson.Marshal(result)
			_, _ = w.Write(bs)
			return
		}
		fieldsMap[field] = struct{}{}
	}

	ctx := r.Context()
	key := fmt.Sprintf("item:%d", id)
	item, err := hf.cache.FindItemValue(ctx, key)
	hf.logger.Info("XXX", zap.String("item", fmt.Sprintf("%v", item)))
	if err == redis.Nil {
		item, err := hf.dbClient.GetAd(id)
		if err != nil && err != gorm.ErrRecordNotFound {
			hf.logger.Error("error during getting data from DB in getAd", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			bs, _ = easyjson.Marshal(result)
			_, _ = w.Write(bs)
			return
		}
		if item != nil {
			itemToCache := item.CreateMap([]string{"description", "image_urls"})
			err = hf.cache.SetItemValue(ctx, key, itemToCache)
			if err != nil {
				hf.logger.Error("error during caching item in getAd", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				bs, _ = easyjson.Marshal(result)
				_, _ = w.Write(bs)
				return
			}
			if len(fieldsMap) == 2 {
				result.Result = &itemToCache
			} else {
				m := item.CreateMapFromFields(fieldsMap)
				result.Result = &m
			}
		}
	}

	result.Status = "success"
	if item != nil {
		if _, ok := fieldsMap["description"]; !ok {
			item.Description = ""
		}
		if _, ok := fieldsMap["image_urls"]; !ok {
			item.ImageURLs = []string{}
		}
		result.Result = item
	}

	bs, _ = easyjson.Marshal(result)
	_, _ = w.Write(bs)
}

// createAd is a function to create ad
//
// Params: title, description, photo_urls, price
// Return: ID of new ad, code of a result
// @Accept  json
// @Description An endpoint to create item
// @Failure 422 {object} entities.CreateAdAnswer{}
// @Failure 500 {object} entities.CreateAdAnswer{}
// @Success 200 {object} entities.CreateAdAnswer{}
// @Router /create_ad [post]
func (hf *HandlerFacade) createAd(w http.ResponseWriter, r *http.Request, bs []byte) {
	result := entities.CreateAdAnswer{ID: nil, Status: "error"}
	var item entities.AdJSONItem
	err := easyjson.Unmarshal(bs, &item)
	if err != nil {
		hf.logger.Error("error during Unmarshal in createAd",
			zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		bs, _ = easyjson.Marshal(result)
		_, _ = w.Write(bs)
		return
	}

	err = hf.validator.Struct(item)
	if err != nil {
		hf.logger.Error("error during validating in createAd", zap.Error(err))
		w.WriteHeader(http.StatusUnprocessableEntity)
		bs, _ = easyjson.Marshal(result)
		_, _ = w.Write(bs)
		return
	}

	id, itm, err := hf.dbClient.CreateAd(item)
	if err != nil {
		hf.logger.Error("error during dbClient.createAd in createAd", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		bs, _ = easyjson.Marshal(result)
		_, _ = w.Write(bs)
		return
	}

	ctx := r.Context()
	key := fmt.Sprintf("item:%d", id)
	itemToCache := itm.CreateMap([]string{"description", "image_urls"})
	err = hf.cache.SetItemValue(ctx, key, itemToCache)
	if err != nil {
		hf.logger.Error("error during caching item in createAd", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		bs, _ = easyjson.Marshal(result)
		_, _ = w.Write(bs)
		return
	}

	result.Status = "success"
	result.ID = &id

	bs, _ = easyjson.Marshal(result)
	_, _ = w.Write(bs)
}
