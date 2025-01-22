package sdpassistor

import (
	"encoding/json"

	"github.com/pingostack/livhub/pkg/gortc/rtcerror"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
)

type Payload struct {
	PayloadType        uint8
	RtxPayloadType     uint8
	Kind               string
	EncodingName       string
	Feedback           []string
	ClockRate          uint32
	Fmtp               string
	EncodingParameters string
}

type PayloadUnion struct {
	Audio []*Payload `json:"audio"`
	Video []*Payload `json:"video"`
	Data  []*Payload `json:"data"`
}

func NewPayloadUnion(sd webrtc.SessionDescription) (pu *PayloadUnion, err error) {
	var parsedSdp *sdp.SessionDescription
	parsedSdp, err = sd.Unmarshal()
	if err != nil {
		return nil, errors.Wrap(rtcerror.ErrSdpUnmarshal, err.Error())
	}

	var payloadUnits []*Payload
	payloadUnits, err = GeneratePayloadUnits(parsedSdp)
	if err != nil {
		return nil, err
	}

	pu = &PayloadUnion{}
	for _, p := range payloadUnits {
		if p.Kind == "audio" {
			pu.Audio = append(pu.Audio, p)
		} else if p.Kind == "video" {
			pu.Video = append(pu.Video, p)
		} else if p.Kind == "data" && pu.Data == nil {
			pu.Data = append(pu.Data, p)
		}
	}

	return pu, nil
}

func (pu *PayloadUnion) String() string {
	jstr, _ := json.Marshal(pu)
	return string(jstr)
}

func (pu *PayloadUnion) HasAudio() bool {
	return len(pu.Audio) > 0
}

func (pu *PayloadUnion) HasVideo() bool {
	return len(pu.Video) > 0
}

func (pu *PayloadUnion) HasData() bool {
	return len(pu.Data) > 0
}
