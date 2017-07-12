package ext

const (
	LIEBIAN_SRV_SUCCESS = iota
)

const (
	GET_URL_STATUS_OK = iota
	GET_URL_STATUS_HAS_EXIST
	GET_URL_STATUS_EXPIRED
)

type GetQRCodeUrlReq struct {
	AppId  string `json:"appId,omitempty"`
	OpenId string `json:"openId,omitempty"`
	Type   int64  `json:"type"`
}

type QRCodeUrlInfo struct {
	Name string `json:"name"`
	Url  string `json:"url"`
	Type int64  `json:"type"`
}

type GetQRCodeUrlRsp struct {
	Status int            `json:"status"`
	Result *QRCodeUrlInfo `json:"result,omitempty"`
}

type GetQRCodeUrlResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg,omitempty"`
	Data GetQRCodeUrlRsp `json:"data,omitempty"`
}
