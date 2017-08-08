package ext

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	
	"github.com/reechou/holmes"
	"github.com/reechou/share-liebian/config"
)

type WeixinxExt struct {
	client *http.Client
	cfg    *config.Config
}

func NewWeixinxExt(cfg *config.Config) *WeixinxExt {
	weixinx := &WeixinxExt{
		client: &http.Client{},
		cfg:    cfg,
	}
	
	return weixinx
}

func (self *WeixinxExt) GetLiebianQrCodeUrl(request *GetLiebianInfoReq) (*LiebianInfo, error) {
	reqBytes, err := json.Marshal(request)
	if err != nil {
		holmes.Error("json encode error: %v", err)
		return nil, err
	}
	
	req, err := http.NewRequest("POST", fmt.Sprintf("http://%s/weixin/get_user_lianbian_pool",
		self.cfg.WeixinxSrv.Host), bytes.NewBuffer(reqBytes))
	if err != nil {
		holmes.Error("http new request error: %v", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := self.client.Do(req)
	if err != nil {
		holmes.Error("http do request error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	rspBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		holmes.Error("ioutil ReadAll error: %v", err)
		return nil, err
	}
	var response GetLiebianInfoRsp
	err = json.Unmarshal(rspBody, &response)
	if err != nil {
		holmes.Error("json decode error: %v [%s]", err, string(rspBody))
		return nil, err
	}
	if response.Code != LIEBIAN_SRV_SUCCESS {
		holmes.Error("get weixinx qircode result code error: %d %v", response.Code, response)
		return nil, fmt.Errorf("get weixinx qircode result code error.")
	}
	
	return &response.Data, nil
}

