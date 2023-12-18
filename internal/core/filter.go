package core

var Filter *filter

type filter struct {
	publishFilters []PublishFilter
	playFilters    []PlayFilter
}

type PublishFilter interface {
	Publish(req *PublishReq) error
}

type PlayFilter interface {
	Play(req *PlayReq) error
}

type PublishFilterFunc func(req *PublishReq) error
type PlayFilterFunc func(req *PlayReq) error

func (f PublishFilterFunc) Publish(req *PublishReq) error {
	return f(req)
}

func (f PlayFilterFunc) Play(req *PlayReq) error {
	return f(req)
}

func init() {
	Filter = &filter{
		publishFilters: []PublishFilter{},
		playFilters:    []PlayFilter{},
	}
}

func (f *filter) AddPublishFilter(publishFilter PublishFilter) {
	f.publishFilters = append(f.publishFilters, publishFilter)
}

func (f *filter) AddPlayFilter(playFilter PlayFilter) {
	f.playFilters = append(f.playFilters, playFilter)
}

func (f *filter) Publish(req *PublishReq) error {
	for _, pf := range f.publishFilters {
		err := pf.Publish(req)
		if err != nil {
			return err
		}
	}

	return nil
}

func (f *filter) Play(req *PlayReq) error {
	for _, pf := range f.playFilters {
		err := pf.Play(req)
		if err != nil {
			return err
		}
	}

	return nil
}

func AddPublishFilter(publishFilter PublishFilter) {
	Filter.AddPublishFilter(publishFilter)
}

func AddPlayFilter(playFilter PlayFilter) {
	Filter.AddPlayFilter(playFilter)
}

func Publish(req *PublishReq) error {
	return Filter.Publish(req)
}

func Play(req *PlayReq) error {
	return Filter.Play(req)
}
