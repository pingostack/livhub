package sdpassistor

import (
	"strconv"
	"strings"

	"github.com/pingostack/neon/pkg/rtclib/rtcerror"
	"github.com/pion/sdp/v3"
	"github.com/pkg/errors"
)

func ParseRtpmapPayloadType(rtpmap string) (uint8, error) {
	// a=rtpmap:<payload type> <encoding name>/<clock rate>[/<encoding parameters>]
	split := strings.Split(rtpmap, " ")
	if len(split) != 2 {
		return 0, rtcerror.ErrInvalidRtpmap
	}

	ptSplit := strings.Split(split[0], ":")
	if len(ptSplit) != 2 {
		return 0, rtcerror.ErrInvalidRtpmap
	}

	ptInt, err := strconv.ParseUint(ptSplit[1], 10, 8)
	if err != nil {
		return 0, rtcerror.ErrInvalidRtpmap
	}

	return uint8(ptInt), nil
}

func GetRtxPayloadType(sd *sdp.SessionDescription, pt uint8) (rtx uint8, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(rtcerror.ErrPanics, "GetRtxPayloadType panic: %v", r)
		}
	}()

	for _, media := range sd.MediaDescriptions {
		for _, attr := range media.Attributes {
			if strings.HasPrefix(attr.String(), "rtpmap:") && strings.Contains(attr.String(), "rtx") {
				rtx, err = ParseRtpmapPayloadType(attr.String())
				if err != nil {
					return 0, err
				}
				continue
			}

			if strings.HasPrefix(attr.String(), "fmtp:") && strings.Contains(attr.String(), "apt="+strconv.Itoa(int(pt))) {
				return rtx, nil
			}
		}
	}

	return 0, rtcerror.ErrNoRtxPayload
}

func GetPayloadTypePriority(kind string, sd *sdp.SessionDescription) (pt uint8, rtx uint8, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(rtcerror.ErrPanics, "GetAudioPayloadTypePriority panic: %v", r)
		}
	}()

	for _, media := range sd.MediaDescriptions {
		if !strings.EqualFold(media.MediaName.Media, kind) {
			continue
		}

		for _, attr := range media.Attributes {
			if strings.HasPrefix(attr.String(), "rtpmap:") {
				pt, err = ParseRtpmapPayloadType(attr.String())
				if err != nil {
					return 0, 0, errors.Wrapf(err, "kind: %s", kind)
				}

				rtx, err = GetRtxPayloadType(sd, pt)
				if err != nil && !errors.Is(err, rtcerror.ErrNoRtxPayload) {
					return 0, 0, errors.Wrapf(err, "kind: %s", kind)
				}

				return pt, rtx, nil
			}
		}
	}

	return 0, 0, errors.Wrapf(rtcerror.ErrNoPayload, "kind: %s", kind)
}
