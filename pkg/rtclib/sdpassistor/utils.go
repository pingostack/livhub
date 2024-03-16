package sdpassistor

import (
	"strconv"
	"strings"

	"github.com/pingostack/neon/pkg/rtclib/rtcerror"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
)

func ParseRtpmap(rtpmap string) (pt uint8, encodingName string, clockRate uint32, encodingParameters string, err error) {
	// a=rtpmap:<payload type> <encoding name>/<clock rate>[/<encoding parameters>]
	split := strings.Split(rtpmap, " ")
	if len(split) != 2 {
		return 0, "", 0, "", rtcerror.ErrInvalidRtpmap
	}

	ptInt, err := strconv.ParseUint(split[0], 10, 8)
	if err != nil {
		return 0, "", 0, "", rtcerror.ErrInvalidRtpmap
	}

	pt = uint8(ptInt)

	encodingSplit := strings.Split(split[1], "/")
	if len(encodingSplit) != 2 && len(encodingSplit) != 3 {
		return 0, "", 0, "", rtcerror.ErrInvalidRtpmap
	}

	encodingName = encodingSplit[0]

	cr := uint64(0)
	cr, err = strconv.ParseUint(encodingSplit[1], 10, 32)
	if err != nil {
		return 0, "", 0, "", rtcerror.ErrInvalidRtpmap
	}

	clockRate = uint32(cr)

	if len(encodingSplit) == 3 {
		encodingParameters = encodingSplit[2]
	}

	return pt, encodingName, clockRate, encodingParameters, nil
}

func ParseRtxPayloadType(fmtpAttri string) (rtx uint8, err error) {
	// 96 apt=100
	split := strings.Split(fmtpAttri, " ")
	if len(split) != 2 {
		return 0, rtcerror.ErrInvalidFmtp
	}

	rtxInt, err := strconv.ParseUint(split[0], 10, 8)
	if err != nil {
		return 0, rtcerror.ErrInvalidFmtp
	}

	rtx = uint8(rtxInt)

	return rtx, nil
}

func GetRtxPayloadType(md *sdp.MediaDescription, pt uint8) (rtx uint8, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(rtcerror.ErrPanics, "GetRtxPayloadType panic: %v", r)
		}
	}()

	for _, attr := range md.Attributes {
		if !strings.HasPrefix(attr.String(), "fmtp:") || !strings.Contains(attr.String(), "apt="+strconv.Itoa(int(pt))) {
			continue
		}

		rtx, err = ParseRtxPayloadType(attr.Value)
		if err != nil {
			return 0, err
		}

		return rtx, nil
	}

	return 0, rtcerror.ErrNoRtxPayload
}

func GeneratePayloadUnit(parsedSdp *sdp.SessionDescription, md *sdp.MediaDescription, pt uint8) (pu *Payload, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(rtcerror.ErrPanics, "GetAudioPayloadTypePriority panic: %v", r)
		}
	}()

	var codec sdp.Codec
	codec, err = parsedSdp.GetCodecForPayloadType(pt)
	if err != nil {
		return nil, errors.Wrap(rtcerror.ErrNoCodecForPT, err.Error())
	}

	pu = &Payload{
		PayloadType:  codec.PayloadType,
		Kind:         md.MediaName.Media,
		EncodingName: codec.Name,
		Feedback:     codec.RTCPFeedback,
		ClockRate:    codec.ClockRate,
		Fmtp:         codec.Fmtp,
	}

	pu.RtxPayloadType, err = GetRtxPayloadType(md, pt)
	if err != nil && !errors.Is(err, rtcerror.ErrNoRtxPayload) {
		return nil, errors.Wrap(err, "in GeneratePayloadUnit")
	}

	return pu, nil
}

func GeneratePayloadUnits(sd *sdp.SessionDescription) (puSlices []*Payload, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(rtcerror.ErrPanics, "GetAudioPayloadTypePriority panic: %v", r)
		}
	}()

	for _, media := range sd.MediaDescriptions {
		for _, attr := range media.Attributes {
			if !strings.HasPrefix(attr.String(), "rtpmap:") {
				continue
			}

			pt := uint8(0)
			encodingName := ""
			pt, encodingName, _, _, err = ParseRtpmap(attr.Value)
			if strings.EqualFold(encodingName, "rtx") {
				continue
			}

			pu := &Payload{}
			pu, err = GeneratePayloadUnit(sd, media, pt)
			if err != nil {
				if errors.Is(err, rtcerror.ErrNoCodecForPT) {
					err = nil
					continue
				}
				return nil, err
			}

			puSlices = append(puSlices, pu)
		}
	}

	return puSlices, nil
}

func GetFirstPayloadType(kind string, sd *sdp.SessionDescription) (pt uint8, rtx uint8, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrapf(rtcerror.ErrPanics, "GetFirstPayloadType panic: %v", r)
		}
	}()

	for _, media := range sd.MediaDescriptions {
		if !strings.EqualFold(media.MediaName.Media, kind) {
			continue
		}

		for _, attr := range media.Attributes {
			if strings.HasPrefix(attr.String(), "rtpmap:") {
				pt = uint8(0)
				encodingName := ""
				pt, encodingName, _, _, err = ParseRtpmap(attr.Value)
				if err != nil {
					return 0, 0, err
				}

				if strings.EqualFold(encodingName, "rtx") {
					continue
				}

				rtx, err = GetRtxPayloadType(media, pt)
				if err != nil && !errors.Is(err, rtcerror.ErrNoRtxPayload) {
					return 0, 0, errors.Wrap(err, "in GetFirstPayloadType")
				}

				return pt, rtx, nil
			}
		}
	}

	return 0, 0, errors.Wrapf(rtcerror.ErrNoPayload, "kind: %s", kind)
}

func GetPayloadStatus(sdpStr string, sdpType webrtc.SDPType) (hasAudio, hasVideo, hasData bool, err error) {
	sd := webrtc.SessionDescription{
		Type: sdpType,
		SDP:  sdpStr,
	}

	var parsedSdp *sdp.SessionDescription
	parsedSdp, err = sd.Unmarshal()
	if err != nil {

		err = errors.Wrap(rtcerror.ErrSdpUnmarshal, err.Error())
		return
	}

	puSlices, err := GeneratePayloadUnits(parsedSdp)
	if err != nil {
		return
	}

	for _, pu := range puSlices {
		if pu.Kind == "audio" {
			hasAudio = true
		} else if pu.Kind == "video" {
			hasVideo = true
		} else if pu.Kind == "data" {
			hasData = true
		}
	}

	return
}
