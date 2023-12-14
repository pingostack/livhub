package pms

type Request struct {
	Version string `json:"version"`
	Method  string `json:"method"`
	Stream  string `json:"stream"`
	Session string `json:"session"`
	Data    struct {
		SDP        string `json:"sdp"`
		MaxBitrate int    `json:"max_bitrate"`
	} `json:"data"`
}

type Response struct {
	Version string `json:"version"`
	Method  string `json:"method"`
	Err     int    `json:"err"`
	ErrMsg  string `json:"err_msg"`
	Session string `json:"session"`
	Data    struct {
		SDP string `json:"sdp"`
	} `json:"data"`
}
