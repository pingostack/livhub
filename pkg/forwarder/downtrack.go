package forwarder

type IDownTrack interface {
	IFrameDestination
	SwitchSpatialLayer(targetLayer int, setAsMax bool) error
	SetSimulcast(isSimulcast bool)
	SetInitialLayers(spatialLayer int, temporalLayer int)
	SetMaxTemporalLayer(temporalLayer int)
	SetMaxSpatialLayer(spatialLayer int)
	TrackID() string
	CurrentSpatialLayer() int
	SwitchSpatialLayerDone(layer int)
}

type DefaultDownTrack struct {
	trackID     string
	isSimulcast bool
}

func NewDefaultDownTrack(trackID string, isSimulcast bool) IDownTrack {
	return &DefaultDownTrack{
		trackID:     trackID,
		isSimulcast: isSimulcast,
	}
}

func (d *DefaultDownTrack) SwitchSpatialLayer(targetLayer int, setAsMax bool) error {
	return nil
}

func (d *DefaultDownTrack) SetSimulcast(isSimulcast bool) {
	d.isSimulcast = isSimulcast
}

func (d *DefaultDownTrack) IsSimulcast() bool {
	return d.isSimulcast
}

func (d *DefaultDownTrack) SetInitialLayers(spatialLayer int, temporalLayer int) {
}

func (d *DefaultDownTrack) SetMaxTemporalLayer(temporalLayer int) {
}

func (d *DefaultDownTrack) SetMaxSpatialLayer(spatialLayer int) {
}

func (d *DefaultDownTrack) TrackID() string {
	return d.trackID
}

func (d *DefaultDownTrack) CurrentSpatialLayer() int {
	return 0
}

func (d *DefaultDownTrack) SwitchSpatialLayerDone(layer int) {

}

func (d *DefaultDownTrack) WriteFrame(frame *Frame) error {
	return nil
}

func (d *DefaultDownTrack) Close() {

}
