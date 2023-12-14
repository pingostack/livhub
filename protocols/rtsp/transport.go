package rtsp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type TransportType int
type RtpProfile int

const (
	TransportTypeTcp TransportType = iota
	TransportTypeUdp
)

const (
	RtpProfileInvalid = iota - 1 // invalid
	RtpProfileAVP
	RtpProfileAVPF
	RtpProfileSAVP
	RtpProfileSAVPF
)

const (
	RtpProfileAVPStr   = "rtp/avp"
	RtpProfileAVPFStr  = "rtp/avpf"
	RtpProfileSAVPStr  = "rtp/savp"
	RtpProfileSAVPFStr = "rtp/savpf"
)

type Transport struct {
	profile      RtpProfile
	ty           TransportType
	interleaveds []int
	unicast      bool
	clientPorts  []int
	serverPorts  []int
	ssrc         int64
	mode         string
}

func NewUdpTransport(profile RtpProfile, clientPorts []int) *Transport {
	return &Transport{
		profile:     profile,
		ty:          TransportTypeUdp,
		unicast:     true,
		clientPorts: clientPorts,
	}
}

func NewTcpTransport(profile RtpProfile, interleaveds []int) *Transport {
	return &Transport{
		profile:      profile,
		ty:           TransportTypeTcp,
		interleaveds: interleaveds,
	}
}

func (r RtpProfile) String() string {
	switch r {
	case RtpProfileAVP:
		return RtpProfileAVPStr
	case RtpProfileAVPF:
		return RtpProfileAVPFStr
	case RtpProfileSAVP:
		return RtpProfileSAVPStr
	case RtpProfileSAVPF:
		return RtpProfileSAVPFStr
	}

	return ""
}

func (r *RtpProfile) Parse(s string) error {
	if strings.Contains(s, RtpProfileAVPStr) {
		*r = RtpProfileAVP
		return nil
	} else if strings.Contains(s, RtpProfileAVPFStr) {
		*r = RtpProfileAVPF
		return nil
	} else if strings.Contains(s, RtpProfileSAVPStr) {
		*r = RtpProfileSAVP
		return nil
	} else if strings.Contains(s, RtpProfileSAVPFStr) {
		*r = RtpProfileSAVPF
		return nil
	}

	return fmt.Errorf("invalid rtp profile %s", s)
}

func (t *Transport) SetSSRC(s int32) {
	t.ssrc = int64(s)
}

func (t *Transport) String() string {
	s := "Transport: "

	s += t.profile.String()

	if t.ty == TransportTypeTcp {
		s += "TCP;"

		s += "interleaved="
		for i, v := range t.interleaveds {
			s += strconv.Itoa(v)
			if i < len(t.interleaveds)-1 {
				s += "-"
			}
		}
	} else {
		s += "UDP;"
		if t.unicast {
			s += "unicast;"
		}

		if len(t.clientPorts) > 0 {
			s += "client_port="
			for i, v := range t.clientPorts {
				s += strconv.Itoa(v)
				if i < len(t.clientPorts)-1 {
					s += "-"
				}
			}
		}

		if len(t.serverPorts) > 0 {
			s += "server_port="
			for i, v := range t.serverPorts {
				s += strconv.Itoa(v)
				if i < len(t.serverPorts)-1 {
					s += "-"
				}
			}
		}
	}

	if t.ssrc != 0 {
		s += fmt.Sprintf(";ssrc=%x", t.ssrc)
	}

	return s
}

func UnmarshalTransport(s string) (*Transport, error) {
	t := &Transport{
		profile:      RtpProfileInvalid,
		ty:           TransportTypeUdp,
		interleaveds: make([]int, 2),
		unicast:      true,
		clientPorts:  make([]int, 2),
		serverPorts:  make([]int, 2),
		ssrc:         0,
	}

	kvs := strings.Split(s, ";")

	for _, p := range kvs {
		var key, val string
		kv := strings.Split(p, "=")
		if len(kv) == 2 {
			key = strings.ToLower(kv[0])
			val = kv[1]

			switch key {
			case "interleaved":
				t.interleaveds = make([]int, 2)
				iv := strings.Split(val, "-")
				if len(iv) == 2 {
					t.interleaveds[0], _ = strconv.Atoi(iv[0])
					t.interleaveds[1], _ = strconv.Atoi(iv[1])

					t.ty = TransportTypeTcp
				}

			case "client_port":
				iv := strings.Split(val, "-")
				if len(iv) == 2 {
					t.clientPorts[0], _ = strconv.Atoi(iv[0])
					t.clientPorts[1], _ = strconv.Atoi(iv[1])

					t.ty = TransportTypeUdp
				}

			case "server_port":
				iv := strings.Split(val, "-")
				if len(iv) == 2 {
					t.clientPorts[0], _ = strconv.Atoi(iv[0])
					t.clientPorts[1], _ = strconv.Atoi(iv[1])

					t.ty = TransportTypeUdp
				}

			case "ssrc":
				t.ssrc, _ = strconv.ParseInt(val, 16, 32)

			case "mode":
				t.mode = val
			}

		} else {
			key = strings.ToLower(p)

			if strings.Contains(key, "rtp/") {
				err := t.profile.Parse(key)
				if err != nil {
					return nil, err
				}
			} else if strings.Contains(key, "unicast") {
				t.unicast = true
			}
		}
	}

	if t.ty == TransportTypeUdp && len(t.clientPorts) != 2 {
		return nil, errors.New("invalid transport ports")
	}

	if t.ty == TransportTypeTcp && len(t.interleaveds) != 2 {
		return nil, errors.New("invalid transport interleaved")
	}

	if t.profile == RtpProfileInvalid {
		return nil, errors.New("invalid rtp profile")
	}

	// if t.ssrc == 0 {
	// 	return nil, errors.New("invalid ssrc")
	// }

	return t, nil
}

func (t *Transport) RtpPort() int {
	if len(t.clientPorts) == 0 {
		return -1
	}

	return t.clientPorts[0]
}

func (t *Transport) RtcpPort() int {
	if len(t.clientPorts) == 0 {
		return -1
	}

	return t.clientPorts[1]
}

func (t *Transport) RtpInterleaved() int {
	if len(t.interleaveds) == 0 {
		return -1
	}

	return t.interleaveds[0]
}

func (t *Transport) RtcpInterleaved() int {
	if len(t.interleaveds) == 0 {
		return -1
	}

	return t.interleaveds[1]
}
