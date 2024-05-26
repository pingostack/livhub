package rtc

import (
	"strconv"

	"github.com/pingostack/neon/pkg/deliver"
	"github.com/pingostack/neon/pkg/rtclib/sdpassistor"
	"github.com/pion/webrtc/v4"
)

func convAudioMetadata(audioPayload *sdpassistor.Payload) *deliver.AudioMetadata {
	if audioPayload == nil {
		return nil
	}

	channels := uint64(0)
	var err error
	if audioPayload.EncodingParameters != "" {
		channels, err = strconv.ParseUint(audioPayload.EncodingParameters, 10, 8)
		if err != nil {
			channels = 0
		}
	}
	return &deliver.AudioMetadata{
		Codec:          audioPayload.EncodingName,
		CodecType:      deliver.ConvCodecType(audioPayload.EncodingName),
		RtpPayloadType: audioPayload.PayloadType,
		SampleRate:     audioPayload.ClockRate,
		Channels:       uint8(channels),
	}
}

func convVideoMetadata(videoPayload *sdpassistor.Payload) *deliver.VideoMetadata {
	if videoPayload == nil {
		return nil
	}

	// TODO: get width and height and fps
	width := 0
	height := 0
	fps := 0

	return &deliver.VideoMetadata{
		Codec:          videoPayload.EncodingName,
		CodecType:      deliver.ConvCodecType(videoPayload.EncodingName),
		RtpPayloadType: videoPayload.PayloadType,
		ClockRate:      videoPayload.ClockRate,
		Width:          width,
		Height:         height,
		FPS:            fps,
	}
}

func convMetadata(pu *sdpassistor.PayloadUnion) deliver.Metadata {
	deliverMd := deliver.Metadata{}
	if pu.HasAudio() {
		deliverMd.Audio = convAudioMetadata(pu.Audio[0])
	}

	if pu.HasVideo() {
		deliverMd.Video = convVideoMetadata(pu.Video[0])
	}

	if pu.HasData() {
		deliverMd.Data = &deliver.DataMetadata{
			Codec: deliver.CodecTypeNone.String(),
		}
	}

	deliverMd.PacketType = deliver.PacketTypeRtp

	return deliverMd
}

func convFormatSettings(pu *sdpassistor.PayloadUnion) deliver.FormatSettings {
	fs := deliver.FormatSettings{}
	for _, p := range pu.Audio {
		fs.AudioCandidates = append(fs.AudioCandidates, *convAudioMetadata(p))
	}

	for _, p := range pu.Video {
		fs.VideoCandidates = append(fs.VideoCandidates, *convVideoMetadata(p))
	}

	if pu.HasData() {
		fs.DataCandidates = append(fs.DataCandidates, deliver.DataMetadata{
			Codec: deliver.CodecTypeNone.String(),
		})
	}

	return fs
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
