package forwarder

type FilterConfig struct {
	filter    IFilter
	inCodec   FrameFormat
	inPacket  PacketType
	outCodec  FrameFormat
	outPacket PacketType
}

type FilterFactory struct {
	filters []FilterConfig
}

func (ff *FilterFactory) Register(inCodec FrameFormat, inPacket PacketType, outCodec FrameFormat, outPacket PacketType, fn func() IFilter) {
	fc := FilterConfig{
		inCodec:   inCodec,
		inPacket:  inPacket,
		outCodec:  outCodec,
		outPacket: outPacket,
		filter:    fn,
	}

	ff.filters = append(ff.filters, fc)
}

func (ff *FilterFactory) Get(inCodec FrameFormat, inPacket PacketType, outCodec FrameFormat, outPacket PacketType) IFilter {
	for _, fc := range ff.filters {
		if fc.inCodec == inCodec && fc.inPacket == inPacket && fc.outCodec == outCodec && fc.outPacket == outPacket {
			return fc.filter
		}
	}

	return nil
}

func (ff *FilterFactory) Filters() []IFilter {
	filters := make([]IFilter, 0)
	for _, fc := range ff.filters {
		filters = append(filters, fc.filter)
	}

	return filters
}

var DefaultFilterFactory = &FilterFactory{
	filters: make([]FilterConfig, 0),
}
