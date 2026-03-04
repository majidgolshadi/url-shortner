package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

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
		URL   string `json:"url"`
		Token string `json:"token"`
	}

	InternalServerError struct {
		Message string `json:"message"`
	}
)

type (
	URLHandler struct {
		urlService URLService
	}
	URLService interface {
		Add(ctx context.Context, url string) (token string, insertError error)
		Delete(ctx context.Context, token string) error
		Fetch(ctx context.Context, token string) (*domain.URL, error)
	}
)

func NewURLHandler(urlService URLService) *URLHandler {
	return &URLHandler{
		urlService: urlService,
	}
}

func (uh *URLHandler) addUrlHandle(resp http.ResponseWriter, req *http.Request) {
	var request AddUrlRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil || request.URL == "" {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	token, err := uh.urlService.Add(req.Context(), request.URL)
	if err != nil {
		uh.internalServerError(err, resp)
		return
	}

	resp.WriteHeader(http.StatusOK)
	// nolint:errcheck
	json.NewEncoder(resp).Encode(&AddUrlResponse{
		Token: token,
	})
}

func (uh *URLHandler) fetchUrlHandle(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	if token == "" {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: check the url owner
	urlData, err := uh.urlService.Fetch(req.Context(), token)
	if err != nil {
		uh.internalServerError(err, resp)
		return
	}

	resp.WriteHeader(http.StatusOK)
	// nolint:errcheck
	json.NewEncoder(resp).Encode(&FetchUrlResponse{
		URL:   urlData.Path,
		Token: urlData.Token,
	})
}

func (uh *URLHandler) deleteUrlHandle(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	if token == "" {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: check the url owner
	err := uh.urlService.Delete(req.Context(), token)
	if err != nil {
		uh.internalServerError(err, resp)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

func (uh *URLHandler) internalServerError(err error, resp http.ResponseWriter) {
	resp.WriteHeader(http.StatusInternalServerError)
	// nolint:errcheck
	json.NewEncoder(resp).Encode(&InternalServerError{
		Message: err.Error(),
	})
}
