package ext

type GetLiebianInfoReq struct {
	LiebianType int64  `json:"liebianType"`
	AppId       string `json:"appId"`
	OpenId      string `json:"openId"`
}

type LiebianInfo struct {
	Status int64  `json:"status"`
	Qrcode string `json:"qrcode"`
}

type GetLiebianInfoRsp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg,omitempty"`
	Data LiebianInfo `json:"data,omitempty"`
}
