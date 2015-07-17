package caspercloud

import (
	"encoding/json"
	"fmt"
	"github.com/BigTong/gocounter"
	"github.com/xlvector/dlog"
	"net/http"
	"net/url"
	"runtime/debug"
)

const (
	kInternalErrorResut = "server get internal result"
)

type CasperServer struct {
	cmdCache   *CommandCache
	ct         *gocounter.Counter
	cmdFactory CommandFactory
}

func NewCasperServer(cf CommandFactory) *CasperServer {
	return &CasperServer{
		cmdCache:   NewCommandCache(),
		ct:         gocounter.NewCounter(),
		cmdFactory: cf,
	}
}

func (self *CasperServer) setArgs(cmd Command, params url.Values) *Output {
	args := self.getArgs(params)
	dlog.Println("setArgs:", args)
	cmd.SetInputArgs(args)

	if message := cmd.GetMessage(); message != nil {
		return message
	}
	return nil
}

func (self *CasperServer) getArgs(params url.Values) map[string]string {
	args := make(map[string]string)
	for k, v := range params {
		args[k] = v[0]
	}
	return args
}

func (self *CasperServer) Process(params url.Values) *Output {
	dlog.Info("%s", params.Encode())
	id := params.Get("id")
	if len(id) == 0 {
		c := self.cmdFactory.CreateCommand(params)
		if c == nil {
			return &Output{Status: FAIL, Data: "no create command"}
		}
		self.cmdCache.SetCommand(c)
		params.Set("id", c.GetId())
		return self.setArgs(c, params)
	}

	dlog.Info("get id:%s", id)
	c := self.cmdCache.GetCommand(id)
	if c == nil {
		dlog.Warn("get nil command id:%s", id)
		return &Output{Status: FAIL, Data: "not get command"}
	}

	dlog.Info("get cmd:%s", id)
	ret := self.setArgs(c, params)

	if c.Finished() {
		c.Successed()
		self.cmdCache.Delete(id)
		return &Output{Status: FINISH_FETCH_DATA}
	}

	if ret.Status == FAIL {
		c.Successed()
		return &Output{Status: FAIL, Data: ret.Data}
	}

	return ret
}

func (self *CasperServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			dlog.Println("ERROR: http submit", r)
			debug.PrintStack()
		}
	}()
	self.ct.Incr("request", 1)
	params := req.URL.Query()
	ret := self.Process(params)
	output, _ := json.Marshal(ret)
	fmt.Fprint(w, string(output))
	return
}
