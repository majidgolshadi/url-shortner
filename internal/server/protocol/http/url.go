package http

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"

	"github.com/majidgolshadi/url-shortner/internal/domain"
)

type (
	AddUrlRequest struct {
		URL string `json:"url"`
	}

	AddUrlResponse struct {
		Token string `json:"token"`
	}

	FetchUrlResponse struct {
		URL string `json:"url"`
		Token string `json:"token"`
	}

	InternalServerError struct {
		Message string `json:"message"`
	}
)

type (
	UrlHandler struct {
		urlService UrlService
	}
	 UrlService interface {
		Add(ctx context.Context, url string) (token string, insertError error)
		Delete(ctx context.Context, token string) error
		Fetch(ctx context.Context, token string) (*domain.Url, error)
	}
)

func NewUrlHandler(urlService UrlService) *UrlHandler {
	return &UrlHandler{
		urlService: urlService,
	}
}

func (uh *UrlHandler) addUrlHandle(resp http.ResponseWriter, req *http.Request) {
	var request AddUrlRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := uh.urlService.Add(req.Context(), request.URL)
	if err != nil {
		uh.internalServerError(err, resp)
		return
	}

	resp.WriteHeader(http.StatusOK)
	json.NewEncoder(resp).Encode(&AddUrlResponse{
		Token: token,
	})
}

func (uh *UrlHandler) fetchUrlHandle(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	// TODO: check the url owner
	urlData, err := uh.urlService.Fetch(req.Context(), token)
	if err != nil {
		uh.internalServerError(err, resp)
		return
	}

	resp.WriteHeader(http.StatusOK)
	json.NewEncoder(resp).Encode(&FetchUrlResponse{
		URL: urlData.UrlPath,
		Token: urlData.UrlPath,
	})
}

func (uh *UrlHandler) deleteUrlHandle(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	// TODO: check the url owner
	err := uh.urlService.Delete(req.Context(), token)
	if err != nil {
		uh.internalServerError(err, resp)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

func (uh *UrlHandler) internalServerError(err error, resp http.ResponseWriter) {
	resp.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(resp).Encode(&InternalServerError{
		Message: err.Error(),
	})
}

