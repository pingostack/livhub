package rtsp

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type MethodEnum int

const (
	UnknownMethod MethodEnum = iota - 1
	OptionsMethod
	DescribeMethod
	AnnounceMethod
	SetupMethod
	PlayMethod
	PauseMethod
	TeardownMethod
	GetParameterMethod
	SetParameterMethod
	RecordMethod
)

type Request struct {
	method  string
	url     string
	version string
	lines   HeaderLines
	content []byte
}

type IRequest interface {
	Url() string
	MethodStr() string
	GetLine(key string) string
	SetLine(key, value string)
	String() string
	CSeq() int
	Session() string
	ContentType() string
	GetContent() []byte
	SetContent(content string)
}

func UnmarshalRequest(buf []byte) (*Request, int, error) {
	headerEndOffset := bytes.Index(buf, []byte("\r\n\r\n"))
	if headerEndOffset == -1 {
		return nil, 0, nil
	}

	endOffset := headerEndOffset + 4

	contentLength := 0

	req := &Request{
		lines: make(HeaderLines),
	}

	lines := bytes.Split(buf[:headerEndOffset], []byte("\r\n"))
	if len(lines) < 2 {
		return nil, endOffset, errors.New("read lines error, invalid packet")
	}

	// parse first line
	methodLine := lines[0]
	methodLineParts := bytes.Split(methodLine, []byte(" "))
	if len(methodLineParts) != 3 {
		return nil, endOffset, fmt.Errorf("invalid method line: %s", string(methodLine))
	}

	req.method = strings.ToLower(string(methodLineParts[0]))
	req.url = strings.ToLower(string(methodLineParts[1]))
	req.version = strings.ToLower(string(methodLineParts[2]))

	// parse other lines
	for _, line := range lines[1:] {
		if len(line) == 0 {
			continue
		}

		idx := bytes.Index(line, []byte(":"))
		if idx == -1 {
			continue
		}

		key := strings.ToLower(string(line[:idx]))
		value := string(line[idx+1:])
		value = strings.TrimSpace(value)
		req.lines[key] = value

		if key == "content-length" {
			var err error
			contentLength, err = strconv.Atoi(value)
			if err != nil {
				return nil, endOffset, err
			}
		}
	}

	if contentLength > len(buf[endOffset:]) {
		return nil, 0, nil
	}

	req.content = make([]byte, 0)

	if contentLength > 0 {
		req.content = append(req.content, buf[endOffset:endOffset+contentLength]...)
		endOffset += contentLength
	}

	return req, endOffset, nil
}

func (lines HeaderLines) String() string {
	s := ""
	for k, v := range lines {
		s += k + ": " + v + "\r\n"
	}
	return s
}

func (req *Request) Method() MethodEnum {
	switch req.method {
	case "options":
		return OptionsMethod
	case "describe":
		return DescribeMethod
	case "announce":
		return AnnounceMethod
	case "setup":
		return SetupMethod
	case "play":
		return PlayMethod
	case "pause":
		return PauseMethod
	case "teardown":
		return TeardownMethod
	case "getparameter":
		return GetParameterMethod
	case "setparameter":
		return SetParameterMethod
	case "record":
		return RecordMethod
	default:
		return UnknownMethod
	}
}

func (req *Request) Url() string {
	return req.url
}

func (req *Request) MethodStr() string {
	return req.method
}

func (req *Request) GetLine(key string) string {
	return req.lines[key]
}

func (req *Request) SetLine(key, value string) {
	req.lines[key] = value
}

func (req *Request) String() string {
	ret := req.method + " " + req.url + " " + req.version + "\r\n" +
		req.lines.String() +
		"\r\n"

	if len(req.content) > 0 {
		ret = ret + string(req.content)
	}

	return ret
}

func (req *Request) CSeq() int {
	cseqLine := req.lines["cseq"]
	if cseqLine == "" {
		return -1
	}

	cseq, err := strconv.Atoi(cseqLine)
	if err != nil {
		return -1
	}

	return cseq
}

func (req *Request) Session() string {
	return req.lines["session"]
}

func (req *Request) ContentType() string {
	return req.lines["content-type"]
}

func (req *Request) SetContent(content string) {
	req.content = []byte(content)
	req.lines["Content-Length"] = strconv.Itoa(len(content))
}

func (req *Request) GetContent() []byte {
	return req.content
}

func (req *Request) Option() *OptionsRequest {
	return &OptionsRequest{
		IRequest: req,
	}
}

func (req *Request) Describe() *DescribeRequest {
	return &DescribeRequest{
		IRequest: req,
	}
}

func (req *Request) Announce() *AnnounceRequest {
	return &AnnounceRequest{
		IRequest: req,
	}
}

func (req *Request) Setup() *SetupRequest {
	return &SetupRequest{
		IRequest: req,
	}
}

func (req *Request) Play() *PlayRequest {
	return &PlayRequest{
		IRequest: req,
	}
}

func (req *Request) Pause() *PauseRequest {
	return &PauseRequest{
		IRequest: req,
	}
}

func (req *Request) Teardown() *TeardownRequest {
	return &TeardownRequest{
		IRequest: req,
	}
}

func (req *Request) GetParameter() *GetParameterRequest {
	return &GetParameterRequest{
		IRequest: req,
	}
}

func (req *Request) SetParameter() *SetParameterRequest {
	return &SetParameterRequest{
		IRequest: req,
	}
}

func (req *Request) Record() *RecordRequest {
	return &RecordRequest{
		IRequest: req,
	}
}

// OptionsRequest is a RTSP OPTIONS request
type OptionsRequest struct {
	IRequest
}

func (req *OptionsRequest) Require() string {
	return req.GetLine("require")
}

func (req *OptionsRequest) ProxyRequire() string {
	return req.GetLine("proxy-require")
}

// DescribeRequest is a RTSP DESCRIBE request
type DescribeRequest struct {
	IRequest
}

func (req *DescribeRequest) Accept() string {
	return req.GetLine("accept")
}

func (req *DescribeRequest) SetSdp(sdp string) {
	req.SetLine("content-type", "application/sdp")
	req.SetContent(sdp)
}

// AnnounceRequest is a RTSP ANNOUNCE request
type AnnounceRequest struct {
	IRequest
}

// SetupRequest is a RTSP SETUP request
type SetupRequest struct {
	IRequest
}

func (req *SetupRequest) TransportString() string {
	return req.GetLine("transport")
}

func (req *SetupRequest) Transport() (*Transport, error) {
	return UnmarshalTransport(req.TransportString())
}

func (req *SetupRequest) SetTransport(transport *Transport) {
	req.SetLine("transport", transport.String())
}

// PlayRequest is a RTSP PLAY request
type PlayRequest struct {
	IRequest
}

func (req *PlayRequest) Range() string {
	return req.GetLine("range")
}

// PauseRequest is a RTSP PAUSE request
type PauseRequest struct {
	IRequest
}

func (req *PauseRequest) Range() string {
	return req.GetLine("range")
}

// TeardownRequest is a RTSP TEARDOWN request
type TeardownRequest struct {
	IRequest
}

// GetParameterRequest is a RTSP GET_PARAMETER request
type GetParameterRequest struct {
	IRequest
}

// SetParameterRequest is a RTSP SET_PARAMETER request
type SetParameterRequest struct {
	IRequest
}

// RecordRequest is a RTSP RECORD request
type RecordRequest struct {
	IRequest
}
