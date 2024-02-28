package sdpassistor

import (
	"fmt"
	"strings"

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
			err = errors.New(fmt.Sprintf("panic: %v", r))
		}
	}()

	var parsedSdp *sdp.SessionDescription
	parsedSdp, err = sd.Unmarshal()
	if err != nil {
		return
	}

	filterAttributes := func(attrs []sdp.Attribute) []sdp.Attribute {
		var filteredAttrs []sdp.Attribute
		for _, attr := range attrs {
			if attr.Key == sdp.AttrKeyCandidate {
				if preferTCP && strings.Contains(attr.Value, "udp") {
					continue
				}
			} else if attr.Key == "fmtp" {
			} else if attr.Key == "rtpmap" {
			} else if attr.Key == "rtcp-fb" {
			} else if attr.Key == "extmap" {
			} else {
				filteredAttrs = append(filteredAttrs, attr)
			}

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
