package controller

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/chanxuehong/rand"
	"github.com/chanxuehong/session"
	mpoauth2 "github.com/chanxuehong/wechat.v2/mp/oauth2"
	"github.com/chanxuehong/wechat.v2/oauth2"
	"github.com/reechou/holmes"
	"github.com/reechou/share-liebian/ext"
	"github.com/reechou/share-liebian/proto"
)

const (
	LePrefix = "/le"
)

const (
	SHARE_URI_RECEIVE = "receive"
	SHARE_URI_SHOW    = "show"
	SHARE_URI_DETAIL  = "detail"
	SHARE_URI_DETAILX = "detailx"
)

type ShareTpl struct {
	Title string
	Img   string
	Ty    int
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
	oauth2Client        *oauth2.Client

	sessionStorage *session.Storage
	oauth2Num      int
	oauth2Endpoint []oauth2.Endpoint
	wxOauth2Client []*oauth2.Client
}

func NewLeHandler(l *Logic) *LeHandler {
	lh := &LeHandler{l: l}

	lh.lefitSessionStorage = session.New(20*60, 60*60)
	lh.lefitOauth2Endpoint = mpoauth2.NewEndpoint(lh.l.cfg.LefitOauth.LefitWxAppId, lh.l.cfg.LefitOauth.LefitWxAppSecret)
	lh.oauth2Client = &oauth2.Client{
		Endpoint: lh.lefitOauth2Endpoint,
	}

	lh.sessionStorage = session.New(20*60, 60*60)
	lh.oauth2Num = len(lh.l.cfg.WxOauth.WxAppId)
	lh.oauth2Endpoint = make([]oauth2.Endpoint, lh.oauth2Num)
	lh.wxOauth2Client = make([]*oauth2.Client, lh.oauth2Num)
	for i := 0; i < lh.oauth2Num; i++ {
		lh.oauth2Endpoint[i] = mpoauth2.NewEndpoint(
			lh.l.cfg.WxOauth.WxAppId[i],
			lh.l.cfg.WxOauth.WxAppSecret[i])
		lh.wxOauth2Client[i] = &oauth2.Client{
			Endpoint: lh.oauth2Endpoint[i],
		}
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
		http.ServeFile(w, r, self.l.cfg.LefitOauth.MpVerifyDir+rr.Path)
		return
	}

	params := strings.Split(rr.Path, "/")
	if len(params) != 2 {
		return
	}

	//holmes.Debug("rr: %v", rr)

	switch params[0] {
	case SHARE_URI_RECEIVE:
		redirectUrl := fmt.Sprintf("%s%s", r.Host, r.URL.String())
		holmes.Debug("start redirectUrl: %s", redirectUrl)

		state := string(rand.NewHex())
		AuthCodeURL := mpoauth2.AuthCodeURL(self.l.cfg.LefitOauth.LefitWxAppId, fmt.Sprintf("%s/%s", self.l.cfg.LefitOauth.LefitOauth2RedirectURI, params[1]), self.l.cfg.LefitOauth.LefitOauth2Scope, state)
		holmes.Debug("auth code url: %s", AuthCodeURL)
		http.Redirect(w, r, AuthCodeURL, http.StatusFound)
	case SHARE_URI_SHOW:
		queryValues, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			io.WriteString(w, err.Error())
			holmes.Error("url parse query error: %v", err)
			return
		}

		code := queryValues.Get("code")
		if code == "" {
			state := string(rand.NewHex())
			redirectUrl := fmt.Sprintf("http://%s%s", r.Host, r.URL.String())
			AuthCodeURL := mpoauth2.AuthCodeURL(self.l.cfg.LefitOauth.LefitWxAppId,
				redirectUrl,
				self.l.cfg.LefitOauth.LefitOauth2Scope, state)
			//holmes.Debug("authCodeURL: %s", AuthCodeURL)
			http.Redirect(w, r, AuthCodeURL, http.StatusFound)
			return
		}

		token, err := self.oauth2Client.ExchangeToken(code)
		if err != nil {
			//io.WriteString(w, "请重新扫描!")
			//holmes.Error("exchange token error: %v", err)
			http.Redirect(w, r, fmt.Sprintf("http://%s%s", r.Host, r.URL.Path), http.StatusFound)
			return
		}
		//holmes.Debug("token: %+v", token)
		lbType, err := strconv.Atoi(params[1])
		if err != nil {
			holmes.Error("strconv param[%s] error: %v", params[1], err)
			return
		}
		//liebianReq := &ext.GetQRCodeUrlReq{
		//	AppId:  self.l.cfg.LefitOauth.LefitWxAppId,
		//	OpenId: token.OpenId,
		//	Type:   int64(lbType),
		//}
		//urlRsp, err := self.l.LiebianExt.GetLiebianQrCodeUrl(liebianReq)
		//if err != nil {
		//	holmes.Error("get lieban qrcode url error: %v", err)
		//	io.WriteString(w, "请再试一次!")
		//	return
		//}
		//if urlRsp.Result.Url == "" {
		//	io.WriteString(w, "暂无二维码可扫!")
		//	return
		//}
		//if self.l.cfg.IfUseRedirect {
		//	if urlRsp.Status == ext.GET_URL_STATUS_EXPIRED {
		//		http.Redirect(w, r, self.l.cfg.ExpiredUrl, http.StatusFound)
		//		return
		//	}
		//}
		// --- new logic
		liebianReq := &ext.GetLiebianInfoReq{
			AppId:       self.l.cfg.LefitOauth.LefitWxAppId,
			OpenId:      token.OpenId,
			LiebianType: int64(lbType),
		}
		urlRsp, err := self.l.weixinxExt.GetLiebianQrCodeUrl(liebianReq)
		if err != nil {
			holmes.Error("get lieban qrcode url of type[%d] error: %v", lbType, err)
			io.WriteString(w, "请再试一次!")
			return
		}
		if urlRsp.Qrcode == "" {
			io.WriteString(w, "暂无二维码可扫, 请稍后重试!")
			return
		}
		if self.l.cfg.IfUseRedirect {
			if urlRsp.Status == ext.GET_URL_STATUS_EXPIRED {
				http.Redirect(w, r, self.l.cfg.ExpiredUrl, http.StatusFound)
				return
			}
		}
		// --- new logic end
		shareData := &ShareTpl{
			Title: "长按扫描二维码",
			Img:   urlRsp.Qrcode,
			Ty:    lbType,
		}
		renderView(w, "./views/share.html", shareData)
		return
	case SHARE_URI_DETAIL:
		queryValues, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			io.WriteString(w, err.Error())
			holmes.Error("url parse query error: %v", err)
			return
		}

		//holmes.Debug("request: %s %s %s", r.URL.Scheme, r.Host, r.URL.Path)
		scheme := "https://"
		//if r.TLS != nil {
		//	scheme = "https://"
		//}

		code := queryValues.Get("code")
		if code == "" {
			state := string(rand.NewHex())
			redirectUrl := fmt.Sprintf("%s%s%s", scheme, r.Host, r.URL.String())
			AuthCodeURL := mpoauth2.AuthCodeURL(self.l.cfg.LefitOauth.LefitWxAppId,
				redirectUrl,
				self.l.cfg.LefitOauth.LefitOauth2Scope, state)
			http.Redirect(w, r, AuthCodeURL, http.StatusFound)
			return
		}

		token, err := self.oauth2Client.ExchangeToken(code)
		if err != nil {
			http.Redirect(w, r, fmt.Sprintf("%s%s%s", scheme, r.Host, r.URL.Path), http.StatusFound)
			return
		}
		lbType, err := strconv.Atoi(params[1])
		if err != nil {
			holmes.Error("strconv param[%s] error: %v", params[1], err)
			return
		}
		// --- new logic
		liebianReq := &ext.GetLiebianInfoReq{
			AppId:       self.l.cfg.LefitOauth.LefitWxAppId,
			OpenId:      token.OpenId,
			LiebianType: int64(lbType),
		}
		urlRsp, err := self.l.weixinxExt.GetLiebianQrCodeUrl(liebianReq)
		if err != nil {
			holmes.Error("get lieban qrcode url of type[%d] error: %v", lbType, err)
			io.WriteString(w, "请再试一次!")
			return
		}
		if urlRsp.Qrcode == "" {
			io.WriteString(w, "暂无二维码可扫, 请稍后重试!")
			return
		}
		if self.l.cfg.IfUseRedirect {
			if urlRsp.Status == ext.GET_URL_STATUS_EXPIRED {
				http.Redirect(w, r, self.l.cfg.ExpiredUrl, http.StatusFound)
				return
			}
		}
		// --- new logic end
		if lbInfo, ok := self.l.cfg.LiebianDetailMap[lbType]; ok {
			renderView(w, "./views/share_detail.html", map[string]interface{}{
				"Title":      lbInfo.Title,
				"HeaderType": lbInfo.HeaderType,
				"Header":     lbInfo.Header,
				"Ab":         lbInfo.Ab,
				"Qrcode":     urlRsp.Qrcode,
				"Ty":         lbType,
			})
		}
		return
	case SHARE_URI_DETAILX:
		queryValues, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			io.WriteString(w, err.Error())
			holmes.Error("url parse query error: %v", err)
			return
		}

		appIdx := 0
		app := queryValues.Get("app")
		if app != "" {
			appIdx, err = strconv.Atoi(app)
			if err != nil {
				holmes.Error("strconv app[%s] error: %v", app, err)
				return
			}
		}
		if appIdx >= self.oauth2Num {
			holmes.Error("app idx[%d] confignum[%d] is not ok", appIdx, self.oauth2Num)
			return
		}

		//holmes.Debug("request: %s %s %s", r.URL.Scheme, r.Host, r.URL.Path)
		scheme := "https://"
		//if r.TLS != nil {
		//	scheme = "https://"
		//}

		code := queryValues.Get("code")
		if code == "" {
			state := string(rand.NewHex())
			redirectUrl := fmt.Sprintf("%s%s%s", scheme, r.Host, r.URL.String())
			AuthCodeURL := mpoauth2.AuthCodeURL(self.l.cfg.WxOauth.WxAppId[appIdx],
				redirectUrl,
				self.l.cfg.WxOauth.Oauth2ScopeBase, state)
			http.Redirect(w, r, AuthCodeURL, http.StatusFound)
			return
		}

		token, err := self.wxOauth2Client[appIdx].ExchangeToken(code)
		if err != nil {
			http.Redirect(w, r, fmt.Sprintf("%s%s%s?app=%d", scheme, r.Host, r.URL.Path, appIdx), http.StatusFound)
			return
		}
		lbType, err := strconv.Atoi(params[1])
		if err != nil {
			holmes.Error("strconv param[%s] error: %v", params[1], err)
			return
		}
		// --- new logic
		liebianReq := &ext.GetLiebianInfoReq{
			AppId:       self.l.cfg.WxOauth.WxAppId[appIdx],
			OpenId:      token.OpenId,
			LiebianType: int64(lbType),
		}
		urlRsp, err := self.l.weixinxExt.GetLiebianQrCodeUrl(liebianReq)
		if err != nil {
			holmes.Error("get lieban qrcode url of type[%d] error: %v", lbType, err)
			io.WriteString(w, "请再试一次!")
			return
		}
		if urlRsp.Qrcode == "" {
			io.WriteString(w, "暂无二维码可扫, 请稍后重试!")
			return
		}
		if self.l.cfg.IfUseRedirect {
			if urlRsp.Status == ext.GET_URL_STATUS_EXPIRED {
				http.Redirect(w, r, self.l.cfg.ExpiredUrl, http.StatusFound)
				return
			}
		}
		holmes.Debug("appidx: %d lbtype: %d qrcode: %s", appIdx, lbType, urlRsp.Qrcode)
		// --- new logic end
		if lbInfo, ok := self.l.cfg.LiebianDetailMap[lbType]; ok {
			renderView(w, "./views/share_detail.html", map[string]interface{}{
				"Title":      lbInfo.Title,
				"HeaderType": lbInfo.HeaderType,
				"Header":     lbInfo.Header,
				"Ab":         lbInfo.Ab,
				"Qrcode":     urlRsp.Qrcode,
				"Ty":         lbType,
			})
		}
		return
	default:
		http.ServeFile(w, r, self.l.cfg.LefitOauth.MpVerifyDir+rr.Path)
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

func renderView(w http.ResponseWriter, tpl string, data interface{}) {
	t, err := template.ParseFiles(tpl)
	if err != nil {
		holmes.Error("parse file error: %v", err)
		return
	}
	err = t.Execute(w, data)
	if err != nil {
		holmes.Error("execute tmp error: %v", err)
		return
	}
}
