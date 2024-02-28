package sdpassistor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pingostack/neon/pkg/rtclib/rtcerror"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
)

type MediaInfo struct {
}

type SdpFilter struct {
	prefersTCP    bool
	firstPriority bool
}

func NewSdpFilter(prefersTCP bool, firstPriority bool) *SdpFilter {
	return &SdpFilter{
		prefersTCP:    prefersTCP,
		firstPriority: firstPriority,
	}
}

func (f *SdpFilter) Filter(sd webrtc.SessionDescription) (result webrtc.SessionDescription, err error) {
	preferTCP := f.prefersTCP
	// videoPayloadType := -1
	// audioPayloadType := -1

	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(rtcerror.ErrPanics, fmt.Sprintf("panic: %v", r))
		}
	}()

	var parsedSdp *sdp.SessionDescription
	parsedSdp, err = sd.Unmarshal()
	if err != nil {
		err = errors.Wrap(rtcerror.ErrSdpUnmarshal, err.Error())
		return
	}

	var payloadTypes []uint8
	var ptAudio, rtxAudio uint8
	ptAudio, rtxAudio, err = GetPayloadTypePriority("audio", parsedSdp)
	if err != nil && !errors.Is(err, rtcerror.ErrNoPayload) {
		err = errors.Wrap(err, "failed to get audio payload type priority")
		return
	} else {
		payloadTypes = append(payloadTypes, ptAudio)
		if rtxAudio != 0 {
			payloadTypes = append(payloadTypes, rtxAudio)
		}
	}

	var ptVideo, rtxVideo uint8
	ptVideo, rtxVideo, err = GetPayloadTypePriority("video", parsedSdp)
	if err != nil && !errors.Is(err, rtcerror.ErrNoPayload) {
		err = errors.Wrap(err, "failed to get video payload type priority")
		return
	} else {
		payloadTypes = append(payloadTypes, ptVideo)
		if rtxVideo != 0 {
			payloadTypes = append(payloadTypes, rtxVideo)
		}
	}

	skipAttribute := func(attrVal string) bool {
		split := strings.Split(attrVal, " ")
		if len(split) == 0 {
			return false
		}

		pt, err := strconv.ParseUint(split[0], 10, 8)
		if err != nil {
			return false
		}

		for _, payloadType := range payloadTypes {
			if uint8(pt) == payloadType {
				return false
			}
		}

		return true
	}

	filterAttributes := func(attrs []sdp.Attribute) []sdp.Attribute {
		var filteredAttrs []sdp.Attribute
		for _, attr := range attrs {
			if attr.Key == sdp.AttrKeyCandidate && preferTCP && strings.Contains(attr.Value, "udp") {
				continue
			}

			if (attr.Key == "fmtp" || attr.Key == "rtpmap" || attr.Key == "rtcp-fb") && skipAttribute(attr.Value) {
				continue
			}

			filteredAttrs = append(filteredAttrs, attr)
		}

		return filteredAttrs
	}

	parsedSdp.Attributes = filterAttributes(parsedSdp.Attributes)
	for _, media := range parsedSdp.MediaDescriptions {
		media.Attributes = filterAttributes(media.Attributes)
	}

	var sdpBytes []byte
	sdpBytes, err = parsedSdp.Marshal()
	if err != nil {
		err = errors.Wrap(err, "failed to marshal sdp")
		return
	}

	result = webrtc.SessionDescription{
		Type: sd.Type,
		SDP:  string(sdpBytes),
	}

	return
}
