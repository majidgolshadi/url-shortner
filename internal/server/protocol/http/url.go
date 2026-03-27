package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intLogger "github.com/majidgolshadi/url-shortner/internal/infrastructure/logger"
)

type (
	AddUrlRequest struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers,omitempty"`
	}

	AddUrlResponse struct {
		Token string `json:"token"`
	}

	FetchUrlResponse struct {
		URL     string            `json:"url"`
		Token   string            `json:"token"`
		Headers map[string]string `json:"headers,omitempty"`
	}

	InternalServerError struct {
		Message string `json:"message"`
	}
)

type (
	URLHandler struct {
		urlService URLService
		logger     *logrus.Entry
	}
	URLService interface {
		Add(ctx context.Context, url string, headers map[string]string) (token string, insertError error)
		Delete(ctx context.Context, token string) error
		Fetch(ctx context.Context, token string) (*domain.URL, error)
	}
)

func NewURLHandler(urlService URLService, logger *logrus.Entry) *URLHandler {
	return &URLHandler{
		urlService: urlService,
		logger:     logger,
	}
}

func (uh *URLHandler) addUrlHandle(resp http.ResponseWriter, req *http.Request) {
	log := intLogger.WithContext(req.Context(), uh.logger).WithField("handler", "add_url")

	var request AddUrlRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil || request.URL == "" {
		log.WithError(err).Warn("invalid request body")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	log = log.WithField("url", request.URL)
	log.Info("processing add URL request")

	token, err := uh.urlService.Add(req.Context(), request.URL, request.Headers)
	if err != nil {
		log.WithError(err).Error("add URL request failed")
		uh.internalServerError(err, resp)
		return
	}

	log.WithField("token", token).Info("add URL request completed")
	resp.WriteHeader(http.StatusOK)
	// nolint:errcheck
	json.NewEncoder(resp).Encode(&AddUrlResponse{
		Token: token,
	})
}

func (uh *URLHandler) fetchUrlHandle(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	log := intLogger.WithContext(req.Context(), uh.logger).WithFields(logrus.Fields{
		"handler": "fetch_url",
		"token":   token,
	})

	if token == "" {
		log.Warn("missing token parameter")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debug("processing fetch URL request")

	// TODO: check the url owner
	urlData, err := uh.urlService.Fetch(req.Context(), token)
	if err != nil {
		log.WithError(err).Error("fetch URL request failed")
		uh.internalServerError(err, resp)
		return
	}

	log.Debug("fetch URL request completed")
	resp.WriteHeader(http.StatusOK)
	// nolint:errcheck
	json.NewEncoder(resp).Encode(&FetchUrlResponse{
		URL:     urlData.Path,
		Token:   urlData.Token,
		Headers: urlData.Headers,
	})
}

func (uh *URLHandler) deleteUrlHandle(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	log := intLogger.WithContext(req.Context(), uh.logger).WithFields(logrus.Fields{
		"handler": "delete_url",
		"token":   token,
	})

	if token == "" {
		log.Warn("missing token parameter")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Info("processing delete URL request")

	// TODO: check the url owner
	err := uh.urlService.Delete(req.Context(), token)
	if err != nil {
		log.WithError(err).Error("delete URL request failed")
		uh.internalServerError(err, resp)
		return
	}

	log.Info("delete URL request completed")
	resp.WriteHeader(http.StatusAccepted)
}

func (uh *URLHandler) redirectHandle(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	log := intLogger.WithContext(req.Context(), uh.logger).WithFields(logrus.Fields{
		"handler": "redirect",
		"token":   token,
	})

	if token == "" {
		log.Warn("missing token parameter")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Debug("processing redirect request")

	urlData, err := uh.urlService.Fetch(req.Context(), token)
	if err != nil {
		log.WithError(err).Error("redirect request failed")
		uh.internalServerError(err, resp)
		return
	}

	// Set custom headers on the redirect response
	for key, value := range urlData.Headers {
		resp.Header().Set(key, value)
	}

	log.WithField("url", urlData.Path).Debug("redirecting to source URL")
	http.Redirect(resp, req, urlData.Path, http.StatusFound)
}

func (uh *URLHandler) internalServerError(err error, resp http.ResponseWriter) {
	resp.WriteHeader(http.StatusInternalServerError)
	// nolint:errcheck
	json.NewEncoder(resp).Encode(&InternalServerError{
		Message: err.Error(),
	})
}