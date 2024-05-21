package rtsp

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"time"
)

type IResponse interface {
	String() string
	CSeq() int
	Session() string
	//	Transport() (*Transport, error)
	ContentLength() int
	Expires() string
	LastModified() string
	Server() string
	Content() []byte
	SetContent(content string)
	Line(key string) string
	SetLine(key, value string)
	Option() *OptionsResponse
}

type Response struct {
	version   string
	status    Status
	statusStr string
	lines     HeaderLines
	content   []byte
}

func NewResponse(cseq int, status Status) *Response {
	resp := &Response{
		version:   "RTSP/1.0",
		status:    status,
		statusStr: status.String(),
		lines: HeaderLines{
			"CSeq":           strconv.Itoa(cseq),
			"Date":           time.Now().Format(time.RFC1123),
			"Content-Length": strconv.Itoa(0),
			"Server":         "Neon-RTSP",
		},
	}

	return resp
}

func (resp *Response) String() string {
	resp.lines["Content-Length"] = strconv.Itoa(len(resp.content))
	return resp.version + " " + strconv.Itoa(int(resp.status)) + " " + resp.status.String() + "\r\n" +
		resp.lines.String() + "\r\n" +
		string(resp.content)
}

func UnmarshalResponse(buf []byte) (*Response, int, error) {
	headerEndOffset := bytes.Index(buf, []byte("\r\n\r\n"))
	if headerEndOffset == -1 {
		return nil, -1, errors.New("incomplete packet")
	}

	endOffset := headerEndOffset + 4

	contentLength := 0

	resp := &Response{
		lines: make(HeaderLines),
	}

	lines := bytes.Split(buf[:headerEndOffset], []byte("\r\n"))
	if len(lines) < 2 {
		return nil, endOffset, errors.New("invalid packet")
	}

	// parse first line
	statusLine := lines[0]
	statusLineParts := bytes.Split(statusLine, []byte(" "))
	if len(statusLineParts) != 3 {
		return nil, endOffset, errors.New("invalid packet")
	}

	resp.version = strings.ToLower(string(statusLineParts[0]))
	status, _ := strconv.Atoi(string(statusLineParts[1]))
	resp.status = Status(status)
	resp.statusStr = string(statusLineParts[2])

	// parse other lines
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		idx := bytes.Index(line, []byte(":"))
		if idx == -1 {
			continue
		}

		key := strings.ToLower(string(line[:idx]))
		value := string(line[idx+1:])
		resp.lines[key] = value

		if key == "content-length" {
			var err error
			contentLength, err = strconv.Atoi(value)
			if err != nil {
				return nil, headerEndOffset + 4, err
			}
		}
	}

	resp.content = buf[headerEndOffset+4 : headerEndOffset+4+contentLength]

	endOffset += contentLength

	return resp, endOffset, nil
}

func (resp *Response) CSeq() int {
	cseqLine := resp.lines["cseq"]
	if cseqLine == "" {
		return -1
	}

	cseq, err := strconv.Atoi(cseqLine)
	if err != nil {
		return -1
	}

	return cseq
}

func (resp *Response) Line(key string) string {
	return resp.lines[key]
}

func (resp *Response) SetLine(key, value string) {
	resp.lines[key] = value
}

func (resp *Response) Session() string {
	return resp.lines["session"]
}

func (resp *Response) Expires() string {
	return resp.lines["expires"]
}

func (resp *Response) LastModified() string {
	return resp.lines["last-modified"]
}

func (resp *Response) Server() string {
	return resp.lines["server"]
}

func (resp *Response) Content() []byte {
	return resp.content
}

func (resp *Response) SetContent(content string) {
	resp.lines["Content-Length"] = strconv.Itoa(len(content))
	resp.content = []byte(content)
}

func (resp *Response) ContentLength() int {
	contentLengthLine := resp.Line("content-length")

	contentLength, err := strconv.Atoi(contentLengthLine)
	if err != nil {
		return -1
	}

	return contentLength
}

func (resp *Response) Option() *OptionsResponse {
	return &OptionsResponse{
		IResponse: resp,
	}
}

func (resp *Response) Describe() *DescribeResponse {
	return &DescribeResponse{
		IResponse: resp,
	}
}

// OptionsResponse is a RTSP OPTIONS request
type OptionsResponse struct {
	IResponse
}

func (resp *OptionsResponse) Public() []string {
	publicLine := resp.Line("public")

	return strings.Split(publicLine, ",")
}

func (resp *OptionsResponse) SetOptions(options []string) {
	resp.SetLine("Public", strings.Join(options, ", "))
}

// DescribeResponse is a RTSP DESCRIBE request
type DescribeResponse struct {
	IResponse
}

func (resp *DescribeResponse) Describe() string {
	return string(resp.Content())
}

func (resp *DescribeResponse) ContentType() string {
	return resp.Line("content-type")
}

func (resp *DescribeResponse) ContentBase() string {
	return resp.Line("content-base")
}

func (resp *DescribeResponse) SetContentType(contentType string) {
	resp.SetLine("content-type", contentType)
}

func (resp *DescribeResponse) SetContentBase(base string) {
	resp.SetLine("content-base", base)
}

type SetupResponse struct {
	IResponse
}

func NewSetupResponse(cseq int, status Status, transport *Transport) *SetupResponse {
	resp := &SetupResponse{
		IResponse: NewResponse(cseq, status),
	}

	resp.SetTransport(transport)

	return resp
}

func (resp *SetupResponse) Transport() (*Transport, error) {
	transportLine := resp.Line("transport")

	return UnmarshalTransport(transportLine)
}

func (resp *SetupResponse) SetTransport(transport *Transport) {
	resp.SetLine("transport", transport.String())
}
