package sdpassistor

import (
	"fmt"
	"strings"

	"github.com/pingostack/neon/pkg/rtclib/rtcerror"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
)

func FilterCandidates(sd webrtc.SessionDescription, preferTCP bool) (result webrtc.SessionDescription, err error) {

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

	// filter attributes and remove unwanted payload types
	filterAttributes := func(attrs []sdp.Attribute) []sdp.Attribute {
		var filteredAttrs []sdp.Attribute
		for _, attr := range attrs {
			if attr.Key == sdp.AttrKeyCandidate && preferTCP && strings.Contains(attr.Value, "udp") {
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

// func FilterMedias(sd webrtc.SessionDescription) (result webrtc.SessionDescription, err error) {

// 	defer func() {
// 		if r := recover(); r != nil {
// 			err = errors.Wrap(rtcerror.ErrPanics, fmt.Sprintf("panic: %v", r))
// 		}
// 	}()

// 	var parsedSdp *sdp.SessionDescription
// 	parsedSdp, err = sd.Unmarshal()
// 	if err != nil {
// 		err = errors.Wrap(rtcerror.ErrSdpUnmarshal, err.Error())
// 		return
// 	}

// 	// reset m line
// 	resetMLine := func(kind string, pt, rtx uint8) {
// 		for i, media := range parsedSdp.MediaDescriptions {
// 			if media.MediaName.Media == kind {
// 				media.MediaName.Formats = []string{strconv.Itoa(int(pt))}
// 				if rtx != 0 {
// 					media.MediaName.Formats = append(media.MediaName.Formats, strconv.Itoa(int(rtx)))
// 				}
// 				parsedSdp.MediaDescriptions[i] = media
// 				break
// 			}
// 		}
// 	}

// 	// check if there is need to filter out unwanted payload types
// 	shouldSkipAttribute := func(attrVal string) bool {
// 		payloadTypes := []uint8{}
// 		payloadTypes = append(payloadTypes, audioPayloadTypes...)
// 		payloadTypes = append(payloadTypes, videoPayloadTypes...)

// 		split := strings.Split(attrVal, " ")
// 		if len(split) == 0 {
// 			return false
// 		}

// 		pt, err := strconv.ParseUint(split[0], 10, 8)
// 		if err != nil {
// 			return false
// 		}

// 		for _, payloadType := range payloadTypes {
// 			if uint8(pt) == payloadType {
// 				return false
// 			}
// 		}

// 		return true
// 	}

// 	// filter attributes and remove unwanted payload types
// 	filterAttributes := func(attrs []sdp.Attribute) []sdp.Attribute {
// 		var filteredAttrs []sdp.Attribute
// 		for _, attr := range attrs {
// 			if attr.Key == sdp.AttrKeyCandidate && preferTCP && strings.Contains(attr.Value, "udp") {
// 				continue
// 			}

// 			if (attr.Key == "fmtp" || attr.Key == "rtpmap" || attr.Key == "rtcp-fb") && firstPriority && shouldSkipAttribute(attr.Value) {
// 				continue
// 			}

// 			filteredAttrs = append(filteredAttrs, attr)
// 		}

// 		return filteredAttrs
// 	}

// 	parsedSdp.Attributes = filterAttributes(parsedSdp.Attributes)
// 	for _, media := range parsedSdp.MediaDescriptions {
// 		media.Attributes = filterAttributes(media.Attributes)
// 	}

// 	var sdpBytes []byte
// 	sdpBytes, err = parsedSdp.Marshal()
// 	if err != nil {
// 		err = errors.Wrap(err, "failed to marshal sdp")
// 		return
// 	}

// 	result = webrtc.SessionDescription{
// 		Type: sd.Type,
// 		SDP:  string(sdpBytes),
// 	}

// 	return
// }
