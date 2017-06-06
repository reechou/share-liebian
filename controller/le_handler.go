package controller

import (
	//"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"html/template"
	"strconv"

	"github.com/chanxuehong/rand"
	"github.com/chanxuehong/session"
	mpoauth2 "github.com/chanxuehong/wechat.v2/mp/oauth2"
	"github.com/chanxuehong/wechat.v2/oauth2"
	"github.com/reechou/holmes"
	"github.com/reechou/share-liebian/proto"
	"github.com/reechou/share-liebian/ext"
	"github.com/chanxuehong/sid"
)

const (
	LePrefix = "/le"
)

const (
	SHARE_URI_RECEIVE = "receive"
	SHARE_URI_SHOW    = "show"
)

type ShareTpl struct {
	Title string
	Img   string
}

type HandlerRequest struct {
	Method string
	Path   string
	Val    []byte
}

type LeHandler struct {
	l *Logic
	
	lefitSessionStorage *session.Storage
	lefitOauth2Endpoint oauth2.Endpoint
	oauth2Client *oauth2.Client
}

func NewLeHandler(l *Logic) *LeHandler {
	lh := &LeHandler{l: l}
	
	lh.lefitSessionStorage = session.New(20*60, 60*60)
	lh.lefitOauth2Endpoint = mpoauth2.NewEndpoint(lh.l.cfg.LefitOauth.LefitWxAppId, lh.l.cfg.LefitOauth.LefitWxAppSecret)
	lh.oauth2Client = &oauth2.Client{
		Endpoint: lh.lefitOauth2Endpoint,
	}
	
	return lh
}

func (self *LeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rr, err := parseRequest(r)
	if err != nil {
		holmes.Error("parse request error: %v", err)
		writeRsp(w, &proto.Response{Code: proto.RESPONSE_ERR})
		return
	}
	
	//holmes.Debug("rr: %v path[%s]", rr, rr.Path)
	if rr.Path == "" {
		//holmes.Debug("in balance heartbeat")
		return
	}
	
	if strings.HasSuffix(rr.Path, "txt") {
		http.ServeFile(w, r, self.l.cfg.LefitOauth.MpVerifyDir + rr.Path)
		return
	}
	
	params := strings.Split(rr.Path, "/")
	if len(params) != 2 {
		return
	}
	
	holmes.Debug("rr: %v", rr)

	switch params[0] {
	case SHARE_URI_RECEIVE:
		sid := sid.New()
		state := string(rand.NewHex())
		if err := self.lefitSessionStorage.Add(sid, state); err != nil {
			io.WriteString(w, err.Error())
			return
		}
		cookie := http.Cookie{
			Name:     "sid",
			Value:    sid,
			HttpOnly: true,
		}
		http.SetCookie(w, &cookie)
		
		redirectUrl := fmt.Sprintf("http://%s%s", r.Host, r.URL.String())
		holmes.Debug("redirectUrl: %s", redirectUrl)
		AuthCodeURL := mpoauth2.AuthCodeURL(self.l.cfg.LefitOauth.LefitWxAppId,
			redirectUrl,
			self.l.cfg.LefitOauth.LefitOauth2Scope, state)
		holmes.Debug("authCodeURL: %s", AuthCodeURL)
		http.Redirect(w, r, AuthCodeURL, http.StatusFound)
	case SHARE_URI_SHOW:
		redirectUrl := fmt.Sprintf("%s%s", r.Host, r.URL.String())
		holmes.Debug("start redirectUrl: %s", redirectUrl)
		
		queryValues, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			io.WriteString(w, err.Error())
			holmes.Error("url parse query error: %v", err)
			return
		}

		code := queryValues.Get("code")
		if code == "" {
			sid := sid.New()
			state := string(rand.NewHex())
			if err := self.lefitSessionStorage.Add(sid, state); err != nil {
				io.WriteString(w, err.Error())
				return
			}
			cookie := http.Cookie{
				Name:     "sid",
				Value:    sid,
				HttpOnly: true,
			}
			http.SetCookie(w, &cookie)
			
			redirectUrl := fmt.Sprintf("http://%s%s", r.Host, r.URL.String())
			holmes.Debug("redirectUrl: %s", redirectUrl)
			AuthCodeURL := mpoauth2.AuthCodeURL(self.l.cfg.LefitOauth.LefitWxAppId,
				redirectUrl,
				self.l.cfg.LefitOauth.LefitOauth2Scope, state)
			holmes.Debug("authCodeURL: %s", AuthCodeURL)
			http.Redirect(w, r, AuthCodeURL, http.StatusFound)
			return
		}
		
		cookie, err := r.Cookie("sid")
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		session, err := self.lefitSessionStorage.Get(cookie.Value)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		savedState := session.(string)
		queryState := queryValues.Get("state")
		if queryState == "" {
			return
		}
		if savedState != queryState {
			str := fmt.Sprintf("state 不匹配, session 中的为 %q, url 传递过来的是 %q", savedState, queryState)
			io.WriteString(w, str)
			return
		}
		
		holmes.Debug("code: %s %s %s %s", code, r.Host, r.URL.String(), r.URL.Scheme)

		token, err := self.oauth2Client.ExchangeToken(code)
		if err != nil {
			io.WriteString(w, "请重新扫描!")
			holmes.Error("exchange token error: %v", err)
			return
		}
		holmes.Debug("token: %+v", token)
		//json.NewEncoder(w).Encode(token)
		lbType, err := strconv.Atoi(params[1])
		if err != nil {
			holmes.Error("strconv param[%s] error: %v", params[1], err)
			return
		}
		liebianReq := &ext.GetQRCodeUrlReq{
			AppId: self.l.cfg.LefitOauth.LefitWxAppId,
			OpenId: token.OpenId,
			Type: int64(lbType),
		}
		imgUrl, err := self.l.LiebianExt.GetLiebianQrCodeUrl(liebianReq)
		if err != nil {
			holmes.Error("get lieban qrcode url error: %v", err)
			io.WriteString(w, "暂无二维码可扫!")
			return
		}
		if imgUrl == "" {
			io.WriteString(w, "暂无二维码可扫!")
			return
		}
		t, err := template.ParseFiles("./views/share.html")
		if err != nil {
			holmes.Error("parse file error: %v", err)
			return
		}
		shareData := &ShareTpl{
			Title: "长按二维码加入",
			Img:   imgUrl,
		}
		err = t.Execute(w, shareData)
		if err != nil {
			holmes.Error("execute tmp error: %v", err)
			return
		}
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
