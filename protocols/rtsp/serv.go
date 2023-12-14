package rtsp

import (
	"time"

	goPool "github.com/panjf2000/gnet/pkg/pool/goroutine"
	"github.com/pion/sdp/v3"
)

const (
	EmptyState State = iota
	OptionsState
	DescribeState
	SetupState
	PlayState
	PauseState
	TeardownState
)

type ServOptions struct {
	IdleTimeout time.Duration `json:"idleTimeout,omitempty" p:"idleTimeout"` // idle timeout
	Logger      Logger
	Write       WriteHandler
}

type Serv struct {
	ss          IServSession
	state       State
	cseqCounter int
	pool        *goPool.Pool
	descChan    chan string
	url         string
	options     ServOptions
	desc        []byte
}

func NewServ(ss IServSession, options ServOptions) *Serv {

	return &Serv{
		ss:          ss,
		state:       EmptyState,
		cseqCounter: 0,
		pool:        goPool.Default(),
		descChan:    make(chan string, 1),
		url:         "",
		options:     options,
	}
}

func (serv *Serv) State() State {
	return serv.state
}

func (serv *Serv) decodeRtpRtcp(buf []byte) (int, error) {
	return 0, nil
}

func (serv *Serv) Feed(buf []byte) (int, error) {
	if len(buf) == 0 {
		serv.Logger().Warnf("rtsp feed empty data")
		return 0, nil
	}

	var err error
	var endOffset int
	if buf[0] != '$' {
		var req *Request
		req, endOffset, err = UnmarshalRequest(buf)
		if err != nil || endOffset == 0 {
			return endOffset, err
		}

		err := serv.handleRequest(req)
		if err != nil {
			return endOffset, err
		}

	} else {
		endOffset, err = serv.decodeRtpRtcp(buf)
		if err != nil {
			return endOffset, err
		}

	}

	return endOffset, nil
}

func (serv *Serv) SetDescribe(desc string) {
	serv.descChan <- desc
}

func (serv *Serv) handleRequest(req *Request) error {
	defer func() {
		if err := recover(); err != nil {
			serv.Logger().Errorf("handleRequest panic: %v", err)
		}
	}()

	serv.pool.Submit(func() {
		defer func() {
			if err := recover(); err != nil {
				serv.Logger().Errorf("handleRequest process panic => req: %v, err: %v", req, err)
			}
		}()

		serv.Logger().Debugf("rtsp request: %s", req.String())

		if serv.url == "" {
			serv.url = req.Url()
		}

		var err error
		switch req.Method() {
		case OptionsMethod:
			err = serv.OptionsProcess(req)
		case DescribeMethod:
			err = serv.DescribeProcess(req)
		case AnnounceMethod:
			err = serv.AnnounceProcess(req)
		case SetupMethod:
			err = serv.SetupProcess(req)
		case PlayMethod:
			err = serv.PlayProcess(req)
		case PauseMethod:
			err = serv.PauseProcess(req)
		case TeardownMethod:
			err = serv.TeardownProcess(req)
		case GetParameterMethod:
			err = serv.GetParameterProcess(req)
		case SetParameterMethod:
			err = serv.SetParameterProcess(req)
		default:
			err = serv.WriteResponseStatus(req.CSeq(), StatusMethodNotAllowed)
		}

		if err != nil {
			serv.Logger().Errorf("rtsp request error: %s", err.Error())
			return
		}
	})
	return nil
}

func (serv *Serv) OptionsProcess(req *Request) error {
	resp := NewResponse(req.CSeq(), StatusOK).Option()
	resp.SetOptions([]string{
		"OPTIONS",
		"ANNOUNCE",
		"DESCRIBE",
		"SETUP",
		"TEARDOWN",
		"PLAY",
		"PAUSE",
		"GET_PARAMETER",
		"SET_PARAMETER",
	})

	return serv.WriteResponse(resp)
}

func (serv *Serv) DescribeProcess(req *Request) error {
	if serv.ss.GetEventListener() != nil {
		if err := serv.ss.GetEventListener().OnDescribe(serv); err != nil {
			serv.Logger().Errorf("rtsp describe error: %s", err.Error())
			return serv.WriteResponseStatus(req.CSeq(), StatusForbidden)
		}
	}

	select {
	case desc := <-serv.descChan:
		serv.Logger().Debugf("rtsp describe get desc: %s", desc)
		resp := NewResponse(req.CSeq(), StatusOK).Describe()
		resp.SetContentType("application/sdp")
		resp.SetContentBase(serv.url)
		resp.SetContent(desc)
		return serv.WriteResponse(resp)

	case <-time.After(serv.options.IdleTimeout * time.Second):
		serv.Logger().Debugf("rtsp describe timeout")
		return serv.WriteResponseStatus(req.CSeq(), StatusNotFound)
	}
}

func (serv *Serv) AnnounceProcess(req *Request) error {
	contentType := req.Announce().ContentType()
	if contentType != "application/sdp" {
		return serv.WriteResponseStatus(req.CSeq(), StatusUnsupportedMediaType)
	}

	serv.desc = req.GetContent()

	var sd sdp.SessionDescription
	if err := sd.Unmarshal([]byte(serv.desc)); err != nil {
		serv.Logger().Errorf("rtsp announce error: %s", err.Error())
		return serv.WriteResponseStatus(req.CSeq(), StatusBadRequest)
	}

	if serv.ss.GetEventListener() != nil {
		if err := serv.ss.GetEventListener().OnAnnounce(serv); err != nil {
			serv.Logger().Errorf("rtsp announce error: %s", err.Error())
			return serv.WriteResponseStatus(req.CSeq(), StatusForbidden)
		}
	}

	return serv.WriteResponse(NewResponse(req.CSeq(), StatusOK))
}

func (serv *Serv) SetupProcess(req *Request) error {
	serv.Logger().Debugf("rtsp setup")
	trans, err := req.Setup().Transport()
	if err != nil {
		serv.Logger().Errorf("rtsp setup error: %s", err)
		return serv.WriteResponseStatus(req.CSeq(), StatusBadRequest)
	}

	serv.Logger().Debugf("rtsp setup transport: %v", *trans)

	return serv.WriteResponse(NewSetupResponse(req.CSeq(), StatusOK, trans))
}

func (serv *Serv) PlayProcess(req *Request) error {
	return nil
}

func (serv *Serv) PauseProcess(req *Request) error {
	return nil
}

func (serv *Serv) TeardownProcess(req *Request) error {
	return nil
}

func (serv *Serv) GetParameterProcess(req *Request) error {
	return nil
}

func (serv *Serv) SetParameterProcess(req *Request) error {
	return nil
}

func (serv *Serv) WriteResponse(resp IResponse) error {
	return serv.options.Write([]byte(resp.String()))
}

func (serv *Serv) WriteResponseStatus(cseq int, status Status) error {
	return serv.WriteResponse(NewResponse(cseq, status))
}

func (serv *Serv) GetDescription() []byte {
	return serv.desc
}

func (serv *Serv) Logger() Logger {
	return serv.options.Logger
}
