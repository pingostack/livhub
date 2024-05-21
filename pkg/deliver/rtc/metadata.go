package rtc

import (
	"strconv"

	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pingostack/neon/pkg/rtclib/sdpassistor"
	"github.com/pion/webrtc/v4"
)

func convMetadata(pu *sdpassistor.PayloadUnion) deliver.Metadata {
	deliverMd := deliver.Metadata{}
	if pu.HasAudio() {
		deliverMd.Audio = &deliver.AudioMetadata{
			Codec:          pu.Audio.EncodingName,
			CodecType:      deliver.ConvCodecType(pu.Audio.EncodingName),
			RtpPayloadType: pu.Audio.PayloadType,
			SampleRate:     pu.Audio.ClockRate,
		}

		if pu.Audio.EncodingParameters != "" {
			channels, _ := strconv.ParseUint(pu.Audio.EncodingParameters, 10, 8)
			deliverMd.Audio.Channels = uint8(channels)
		}
	}

	if pu.HasVideo() {
		deliverMd.Video = &deliver.VideoMetadata{
			Codec:          pu.Video.EncodingName,
			CodecType:      deliver.ConvCodecType(pu.Video.EncodingName),
			RtpPayloadType: pu.Video.PayloadType,
			ClockRate:      pu.Video.ClockRate,
		}
	}

	if pu.HasData() {
		deliverMd.Data = &deliver.DataMetadata{
			Codec: deliver.CodecTypeNone.String(),
		}
	}

	deliverMd.PacketType = deliver.PacketTypeRtp

	return deliverMd
}

func NewMetadataFromSDP(sdpStr string) (deliver.Metadata, error) {
	var sd webrtc.SessionDescription

	sd.SDP = sdpStr
	sd.Type = webrtc.SDPTypeOffer

	pu, err := sdpassistor.NewPayloadUnion(sd)
	if err != nil {
		return deliver.Metadata{}, err
	}

	return convMetadata(pu), nil
}
