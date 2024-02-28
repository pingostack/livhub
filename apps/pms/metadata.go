package pms

import (
	"fmt"

	"github.com/pion/sdp/v3"
)

type VideoMetadata struct {
	Width   int    `json:"width" mapstructure:"width" yaml:"width"`
	Height  int    `json:"height" mapstructure:"height" yaml:"height"`
	Payload int    `json:"payload" mapstructure:"payload" yaml:"payload"`
	SSRC    uint64 `json:"ssrc" mapstructure:"ssrc" yaml:"ssrc"`
	RTX     struct {
		Payload int    `json:"payload" mapstructure:"payload" yaml:"payload"`
		SSRC    uint64 `json:"ssrc" mapstructure:"ssrc" yaml:"ssrc"`
	} `json:"rtx" mapstructure:"rtx" yaml:"rtx"`
}

type AudioMetadata struct {
	Payload int    `json:"payload" mapstructure:"payload" yaml:"payload"`
	SSRC    uint64 `json:"ssrc" mapstructure:"ssrc" yaml:"ssrc"`
}

type DataMetadata struct {
}

type Metadata struct {
	Video *VideoMetadata `json:"video" mapstructure:"video" yaml:"video"`
	Audio *AudioMetadata `json:"audio" mapstructure:"audio" yaml:"audio"`
	Data  *DataMetadata  `json:"data" mapstructure:"data" yaml:"data"`
}

func (m *Metadata) IsEmpty() bool {
	return m.Video == nil && m.Audio == nil && m.Data == nil
}

func (m *Metadata) HasAudio() bool {
	return m.Audio != nil
}

func (m *Metadata) HasVideo() bool {
	return m.Video != nil
}

func (m *Metadata) HasData() bool {
	return m.Data != nil
}

func (m *Metadata) HasRTX() bool {
	return m.Video != nil && m.Video.RTX.Payload != 0
}

func (m *Metadata) ParseSdp(sdStr string) error {

	sd := &sdp.SessionDescription{}
	if err := sd.Unmarshal([]byte(sdStr)); err != nil {
		return err
	}
	for _, mediaDesc := range sd.MediaDescriptions {
		switch mediaDesc.MediaName.Media {
		case "audio":
			m.Audio = &AudioMetadata{}
			if err := m.Audio.parse(mediaDesc); err != nil {
				return err
			}
		case "video":
			m.Video = &VideoMetadata{}
			if err := m.Video.parse(mediaDesc); err != nil {
				return err
			}
		case "application":
			m.Data = &DataMetadata{}
		}
	}

	return nil
}

func (m *AudioMetadata) parse(mediaDesc *sdp.MediaDescription) error {
	for _, attr := range mediaDesc.Attributes {
		switch attr.Key {
		case "rtpmap":
			if err := m.parseRtpmap(attr.Value); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *AudioMetadata) parseRtpmap(rtpmap string) error {
	var payload int
	var codec string
	if _, err := fmt.Sscanf(rtpmap, "%d %s", &payload, &codec); err != nil {
		return err
	}

	m.Payload = payload

	return nil
}

func (m *VideoMetadata) parse(mediaDesc *sdp.MediaDescription) error {
	for _, attr := range mediaDesc.Attributes {
		switch attr.Key {
		case "rtpmap":
			if err := m.parseRtpmap(attr.Value); err != nil {
				return err
			}
		case "rtcp-fb":
			if err := m.parseRtcpFb(attr.Value); err != nil {
				return err
			}
		case "ssrc":
			if err := m.parseSsrc(attr.Value); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *VideoMetadata) parseRtpmap(rtpmap string) error {
	var payload int
	var codec string
	if _, err := fmt.Sscanf(rtpmap, "%d %s", &payload, &codec); err != nil {
		return err
	}

	m.Payload = payload

	return nil
}

func (m *VideoMetadata) parseRtcpFb(rtcpFb string) error {
	var payload int
	var param string
	if _, err := fmt.Sscanf(rtcpFb, "%d %s", &payload, &param); err != nil {
		return err
	}

	if param == "rtx" {
		m.RTX.Payload = payload
	}

	return nil
}

func (m *VideoMetadata) parseSsrc(ssrc string) error {
	var ssrcValue uint64
	if _, err := fmt.Sscanf(ssrc, "%d", &ssrcValue); err != nil {
		return err
	}

	m.SSRC = ssrcValue

	return nil
}

func (m *Metadata) String() string {
	return fmt.Sprintf("video: %v, audio: %v, data: %v", m.Video, m.Audio, m.Data)
}

func (a *AudioMetadata) String() string {
	return fmt.Sprintf("payload: %d, ssrc: %d", a.Payload, a.SSRC)
}

func (v *VideoMetadata) String() string {
	return fmt.Sprintf("payload: %d, ssrc: %d, rtx: %v", v.Payload, v.SSRC, v.RTX)
}
