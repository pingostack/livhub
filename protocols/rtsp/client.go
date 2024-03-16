package rtsp

import (
	"strconv"

	goPool "github.com/panjf2000/gnet/pkg/pool/goroutine"
	"github.com/sirupsen/logrus"
)

type Client struct {
	state       State
	cseqCounter int
	pool        *goPool.Pool
	Url         string
	Write       WriteHandler
}

func NewClient(write WriteHandler) *Client {

	return &Client{
		state:       EmptyState,
		cseqCounter: 0,
		pool:        goPool.Default(),
		Write:       write,
	}
}

func (c *Client) SetUrl(url string) {
	c.Url = url
}

func (c *Client) State() State {
	return c.state
}

func (c *Client) decodeRtpRtcp(buf []byte) (int, error) {
	return 0, nil
}

func (c *Client) Feed(buf []byte) (int, error) {

	if len(buf) == 0 {
		logrus.Warningf("rtsp feed empty data")
		return 0, nil
	}

	var err error
	var endOffset int
	if buf[0] != '$' {
		_, endOffset, err = UnmarshalResponse(buf)
		if err != nil {
			return endOffset, err
		}

		// err = c.responseProcess(resp)
		// if err != nil {
		// 	return endOffset, err
		// }
	} else {
		endOffset, err = c.decodeRtpRtcp(buf)
		if err != nil {
			return endOffset, err
		}

	}

	return endOffset, nil
}

func (c *Client) nextCSeq() string {
	c.cseqCounter++
	return strconv.Itoa(c.cseqCounter)
}

func (c *Client) NewRequest(method string) *Request {
	req := &Request{
		method:  method,
		url:     "*",
		version: "RTSP/1.0",
		lines: HeaderLines{
			"CSeq": c.nextCSeq(),
		},
	}

	return req
}

func (c *Client) NewOptionsRequest() *OptionsRequest {
	req := c.NewRequest("OPTIONS").Option()
	return req
}

func (c *Client) NewDescribeRequest(sdp string) *DescribeRequest {
	req := c.NewRequest("DESCRIBE").Describe()
	req.SetSdp(sdp)
	return req
}

func (c *Client) NewAnnounceRequest() *AnnounceRequest {
	req := c.NewRequest("ANNOUNCE").Announce()
	return req
}

func (c *Client) NewSetupRequest(trackID int, transport *Transport) *SetupRequest {
	req := c.NewRequest("SETUP").Setup()
	req.SetTransport(transport)

	return req
}

func (c *Client) NewPlayRequest() *PlayRequest {
	req := c.NewRequest("PLAY").Play()
	return req
}

func (c *Client) NewPauseRequest() *PauseRequest {
	req := c.NewRequest("PAUSE").Pause()
	return req
}

func (c *Client) NewTeardownRequest() *TeardownRequest {
	req := c.NewRequest("TEARDOWN").Teardown()
	return req
}

func (c *Client) NewGetParameterRequest() *GetParameterRequest {
	req := c.NewRequest("GET_PARAMETER").GetParameter()
	return req
}

func (c *Client) NewSetParameterRequest() *SetParameterRequest {
	req := c.NewRequest("SET_PARAMETER").SetParameter()
	return req
}

func (c *Client) NewRecordRequest() *RecordRequest {
	req := c.NewRequest("RECORD").Record()
	return req
}
