package web

import (
	"fmt"
	"github.com/webitel/storage/app"
	"github.com/webitel/storage/mlog"
	"github.com/webitel/storage/model"
	"github.com/webitel/storage/utils"
	"net/http"
)

type Handler struct {
	App            *app.App
	HandleFunc     func(*Context, http.ResponseWriter, *http.Request)
	RequireSession bool
	TrustRequester bool
	RequireMfa     bool
	IsStatic       bool
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mlog.Debug(fmt.Sprintf("%v - %v", r.Method, r.URL.Path))

	c := &Context{}
	c.App = h.App
	c.T, _ = utils.GetTranslationsAndLocale(w, r)
	c.Params = ParamsFromRequest(r)
	c.RequestId = model.NewId()
	c.IpAddress = utils.GetIpAddress(r)
	c.Path = r.URL.Path
	c.Log = c.App.Log

	token, _ := app.ParseAuthTokenFromRequest(r)

	w.Header().Set(model.HEADER_REQUEST_ID, c.RequestId)
	w.Header().Set("Content-Type", "application/json")
	if r.Method == "GET" {
		w.Header().Set("Expires", "0")
	}

	//TODO
	if len(token) != 0 && h.RequireSession {
		session, err := c.App.GetSession(token)
		if err != nil {
			c.Log.Info("Invalid session", mlog.Err(err))
			if err.StatusCode == http.StatusInternalServerError {
				c.Err = err
			} else {
				c.Err = model.NewAppError("ServeHTTP", "api.context.session_expired.app_error", nil, "token="+token, http.StatusUnauthorized)
			}
		} else {
			c.Session = *session
		}
	}

	c.Log = c.App.Log.With(
		mlog.String("path", c.Path),
		mlog.String("request_id", c.RequestId),
		mlog.String("ip_addr", c.IpAddress),
		mlog.String("user_id", c.Session.UserId),
		mlog.String("method", r.Method),
	)

	if c.Err == nil && h.RequireSession {
		c.SessionRequired()
	}

	if c.Err == nil {
		h.HandleFunc(c, w, r)
	}

	// Handle errors that have occurred
	if c.Err != nil {
		c.Err.Translate(c.T)
		c.Err.RequestId = c.RequestId

		if c.Err.Id == "api.context.session_expired.app_error" {
			c.LogInfo(c.Err)
		} else {
			c.LogError(c.Err)
		}

		c.Err.Where = r.URL.Path

		w.WriteHeader(c.Err.StatusCode)
		w.Write([]byte(c.Err.ToJson()))
	}
}
