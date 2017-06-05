package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/chanxuehong/rand"
	"github.com/chanxuehong/session"
	"github.com/chanxuehong/sid"
	mpoauth2 "github.com/chanxuehong/wechat.v2/mp/oauth2"
	"github.com/chanxuehong/wechat.v2/oauth2"
	"github.com/reechou/holmes"
	"github.com/reechou/share-liebian/proto"
)

const (
	LePrefix = "/le"
)

const (
	SHARE_URI_RECEIVE = "receive"
	SHARE_URI_SHOW    = "show"
)

type HandlerRequest struct {
	Method string
	Path   string
	Val    []byte
}

type LeHandler struct {
	l *Logic
	
	lefitSessionStorage *session.Storage
	lefitOauth2Endpoint oauth2.Endpoint
}

func NewLeHandler(l *Logic) *LeHandler {
	lh := &LeHandler{l: l}
	
	lh.lefitSessionStorage = session.New(20*60, 60*60)
	lh.lefitOauth2Endpoint = mpoauth2.NewEndpoint(lh.l.cfg.LefitOauth.LefitWxAppId, lh.l.cfg.LefitOauth.LefitWxAppSecret)
	
	return lh
}

func (self *LeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rr, err := parseRequest(r)
	if err != nil {
		holmes.Error("parse request error: %v", err)
		writeRsp(w, &proto.Response{Code: proto.RESPONSE_ERR})
		return
	}
	
	holmes.Debug("rr: %v", rr)

	switch rr.Path {
	case SHARE_URI_RECEIVE:
		sid := sid.New()
		state := string(rand.NewHex())

		if err := self.lefitSessionStorage.Add(sid, state); err != nil {
			io.WriteString(w, err.Error())
			holmes.Error("session write error: %v", err)
			return
		}

		cookie := http.Cookie{
			Name:     "sid",
			Value:    sid,
			HttpOnly: true,
		}
		http.SetCookie(w, &cookie)

		AuthCodeURL := mpoauth2.AuthCodeURL(self.l.cfg.LefitOauth.LefitWxAppId, self.l.cfg.LefitOauth.LefitOauth2RedirectURI, self.l.cfg.LefitOauth.LefitOauth2Scope, state)
		holmes.Debug("auth code url: %s", AuthCodeURL)

		http.Redirect(w, r, AuthCodeURL, http.StatusFound)
	case SHARE_URI_SHOW:
		cookie, err := r.Cookie("sid")
		if err != nil {
			io.WriteString(w, err.Error())
			holmes.Error("get cookie error: %v", err)
			return
		}

		session, err := self.lefitSessionStorage.Get(cookie.Value)
		if err != nil {
			io.WriteString(w, err.Error())
			holmes.Error("session get error: %v", err)
			return
		}

		savedState := session.(string) // 一般是要序列化的, 这里保存在内存所以可以这么做
		queryValues, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			io.WriteString(w, err.Error())
			holmes.Error("url parse query error: %v", err)
			return
		}

		code := queryValues.Get("code")
		if code == "" {
			holmes.Error("用户禁止授权")
			return
		}

		queryState := queryValues.Get("state")
		if queryState == "" {
			holmes.Error("state 参数为空")
			return
		}
		if savedState != queryState {
			str := fmt.Sprintf("state 不匹配, session 中的为 %q, url 传递过来的是 %q", savedState, queryState)
			io.WriteString(w, str)
			holmes.Error(str)
			return
		}

		oauth2Client := oauth2.Client{
			Endpoint: self.lefitOauth2Endpoint,
		}
		token, err := oauth2Client.ExchangeToken(code)
		if err != nil {
			io.WriteString(w, err.Error())
			holmes.Error("exchange token error: %v", err)
			return
		}
		holmes.Debug("token: %+v", token)

		json.NewEncoder(w).Encode(token)
	default:
		http.ServeFile(w, r, self.l.cfg.LefitOauth.MpVerifyDir + rr.Path)
	}
}

func parseRequest(r *http.Request) (*HandlerRequest, error) {
	req := &HandlerRequest{}
	req.Path = r.URL.Path[len(LePrefix)+1:]

	result, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return req, errors.New("parse request read error")
	}
	r.Body.Close()

	req.Method = r.Method
	req.Val = result

	return req, nil
}

func writeRsp(w http.ResponseWriter, rsp *proto.Response) {
	w.Header().Set("Content-Type", "application/json")

	if rsp != nil {
		WriteJSON(w, http.StatusOK, rsp)
	}
}
