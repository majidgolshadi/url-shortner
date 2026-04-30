package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/majidgolshadi/url-shortner/internal/domain"
	intErr "github.com/majidgolshadi/url-shortner/internal/infrastructure/errors"
	intLogger "github.com/majidgolshadi/url-shortner/internal/infrastructure/logger"
	"github.com/majidgolshadi/url-shortner/internal/opengraph"
	"github.com/majidgolshadi/url-shortner/internal/server/protocol/http/middleware"
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
		Add(ctx context.Context, url string, headers map[string]string, customerID string) (token string, insertError error)
		Delete(ctx context.Context, token string) error
		Fetch(ctx context.Context, token string) (*domain.URL, error)
		RefreshOG(ctx context.Context, token string) error
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

	customer, ok := middleware.CustomerFromContext(req.Context())
	if !ok {
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	var request AddUrlRequest
	err := json.NewDecoder(req.Body).Decode(&request)
	if err != nil || request.URL == "" {
		log.WithError(err).Warn("invalid request body")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	log = log.WithField("url", request.URL)
	log.Info("processing add URL request")

	token, err := uh.urlService.Add(req.Context(), request.URL, request.Headers, customer.ID)
	if err != nil {
		if intErr.BudgetExceededErr.Is(err) {
			resp.WriteHeader(http.StatusPaymentRequired)
			// nolint:errcheck
			json.NewEncoder(resp).Encode(map[string]string{"message": "budget exceeded"})
			return
		}
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

	customer, ok := middleware.CustomerFromContext(req.Context())
	if !ok {
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Debug("processing fetch URL request")

	urlData, err := uh.urlService.Fetch(req.Context(), token)
	if err != nil {
		log.WithError(err).Error("fetch URL request failed")
		uh.internalServerError(err, resp)
		return
	}

	if urlData.CustomerID != customer.ID {
		log.Warn("customer does not own this URL")
		resp.WriteHeader(http.StatusForbidden)
		// nolint:errcheck
		json.NewEncoder(resp).Encode(map[string]string{"message": "forbidden"})
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

	customer, ok := middleware.CustomerFromContext(req.Context())
	if !ok {
		resp.WriteHeader(http.StatusUnauthorized)
		return
	}

	log.Info("processing delete URL request")

	urlData, err := uh.urlService.Fetch(req.Context(), token)
	if err != nil {
		log.WithError(err).Error("delete URL request failed (fetch)")
		uh.internalServerError(err, resp)
		return
	}

	if urlData.CustomerID != customer.ID {
		log.Warn("customer does not own this URL")
		resp.WriteHeader(http.StatusForbidden)
		// nolint:errcheck
		json.NewEncoder(resp).Encode(map[string]string{"message": "forbidden"})
		return
	}

	err = uh.urlService.Delete(req.Context(), token)
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

	// Bots often do not follow redirects; serving OG HTML directly ensures link previews work.
	userAgent := req.UserAgent()
	if opengraph.IsBotRequest(userAgent) && urlData.OgHTML != "" {
		log.WithField("user_agent", userAgent).Debug("serving OG metadata to bot")
		uh.serveBotResponse(resp, urlData)
		return
	}

	// Set custom headers on the redirect response
	for key, value := range urlData.Headers {
		resp.Header().Set(key, value)
	}

	// 302 (not 301) so browsers re-validate each visit; 301 would be cached and block URL updates.
	log.WithField("url", urlData.Path).Debug("redirecting to source URL")
	http.Redirect(resp, req, urlData.Path, http.StatusFound)
}

func (uh *URLHandler) refreshOgHandle(resp http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	token := vars["token"]

	log := intLogger.WithContext(req.Context(), uh.logger).WithFields(logrus.Fields{
		"handler": "refresh_og",
		"token":   token,
	})

	if token == "" {
		log.Warn("missing token parameter")
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	log.Info("processing OG refresh request")

	err := uh.urlService.RefreshOG(req.Context(), token)
	if err != nil {
		log.WithError(err).Error("OG refresh request failed")
		uh.internalServerError(err, resp)
		return
	}

	log.Info("OG refresh request completed")
	resp.WriteHeader(http.StatusAccepted)
}

// serveBotResponse serves an HTML page with Open Graph meta tags for bot crawlers.
// The page includes a meta refresh and JavaScript redirect to the original URL.
func (uh *URLHandler) serveBotResponse(resp http.ResponseWriter, urlData *domain.URL) {
	resp.Header().Set("Content-Type", "text/html; charset=utf-8")
	resp.WriteHeader(http.StatusOK)

	htmlPage := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
%s
<meta http-equiv="refresh" content="0;url=%s" />
<link rel="canonical" href="%s" />
</head>
<body>
<p>Redirecting to <a href="%s">%s</a></p>
</body>
</html>`, urlData.OgHTML, urlData.Path, urlData.Path, urlData.Path, urlData.Path)

	// nolint:errcheck
	resp.Write([]byte(htmlPage))
}

func (uh *URLHandler) internalServerError(err error, resp http.ResponseWriter) {
	resp.WriteHeader(http.StatusInternalServerError)
	// nolint:errcheck
	json.NewEncoder(resp).Encode(&InternalServerError{
		Message: err.Error(),
	})
}