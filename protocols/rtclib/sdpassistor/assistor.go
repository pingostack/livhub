package sdpassistor

import (
	"errors"
	"strconv"
	"strings"

	"github.com/pion/sdp/v3"
)

func parseRtpmap(rtpmap string) (uint8, error) {
	// a=rtpmap:<payload type> <encoding name>/<clock rate>[/<encoding parameters>]
	split := strings.Split(rtpmap, " ")
	if len(split) != 2 {
		return 0, errors.New("invalid rtpmap")
	}

	ptSplit := strings.Split(split[0], ":")
	if len(ptSplit) != 2 {
		return 0, errors.New("invalid rtpmap")
	}

	ptInt, err := strconv.ParseUint(ptSplit[1], 10, 8)
	if err != nil {
		return 0, errors.New("invalid rtpmap")
	}

	return uint8(ptInt), nil
}

func GetAudioPayloadTypePriority(sd *sdp.SessionDescription) (int, error) {
	for _, media := range sd.MediaDescriptions {
		if !strings.EqualFold(media.MediaName.Media, "audio") {
			continue
		}

		for _, attr := range media.Attributes {
			if strings.HasPrefix(attr.String(), "rtpmap:") {
				pt, err := parseRtpmap(attr.String())
				if err != nil {
					continue
				}

				return int(pt), nil
			}
		}
	}

	return 0, errors.New("no audio payload type found")
}

func GetVideoPayloadTypePriority(sd *sdp.SessionDescription) (int, error) {
	for _, media := range sd.MediaDescriptions {
		if !strings.EqualFold(media.MediaName.Media, "video") {
			continue
		}

		for _, attr := range media.Attributes {
			if strings.HasPrefix(attr.String(), "rtpmap:") {
				pt, err := parseRtpmap(attr.String())
				if err != nil {
					continue
				}

				return int(pt), nil
			}
		}
	}

	return 0, errors.New("no video payload type found")
}
