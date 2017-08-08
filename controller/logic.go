package controller

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/reechou/holmes"
	"github.com/reechou/share-liebian/config"
	"github.com/reechou/share-liebian/ext"
)

type Logic struct {
	sync.Mutex
	
	LiebianExt *ext.LiebianExt
	weixinxExt *ext.WeixinxExt
	
	cfg *config.Config
}

func NewLogic(cfg *config.Config) *Logic {
	l := &Logic{
		cfg: cfg,
	}
	l.LiebianExt = ext.NewLiebianExt(cfg)
	l.weixinxExt = ext.NewWeixinxExt(cfg)
	l.init()

	return l
}

func (self *Logic) init() {
	http.HandleFunc("/robot/receive_msg", self.RobotReceiveMsg)
}

func (self *Logic) Run() {
	defer holmes.Start(holmes.LogFilePath("./log"),
		holmes.EveryDay,
		holmes.AlsoStdout,
		holmes.DebugLevel).Stop()

	if self.cfg.Debug {
		EnableDebug()
	}
	
	mux := http.NewServeMux()
	mux.Handle("/", &IndexHandler{})
	mux.Handle(LePrefix+"/", NewLeHandler(self))

	holmes.Info("server starting on[%s] ...", self.cfg.Host)
	holmes.Infoln(http.ListenAndServe(self.cfg.Host, mux))
}

func WriteJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "x-requested-with,content-type")
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

func WriteBytes(w http.ResponseWriter, code int, v []byte) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "x-requested-with,content-type")
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(code)
	w.Write(v)
}

func EnableDebug() {
	holmes.Info("server start with debug..")
}
