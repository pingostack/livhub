/*
 * Copy from v2ray-core
 * Modified by pingostack
 */

package plugo

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Feature interface {
	Type() interface{}
	Start() error
	Close() error
}

type resolution struct {
	deps     []reflect.Type
	callback interface{}
}

func getFeature(allFeatures []Feature, t reflect.Type) Feature {
	for _, f := range allFeatures {
		if reflect.TypeOf(f.Type()) == t {
			return f
		}
	}
	return nil
}

func (r *resolution) resolve(allFeatures []Feature) (bool, error) {
	var fs []Feature
	for _, d := range r.deps {
		f := getFeature(allFeatures, d)
		if f == nil {
			return false, nil
		}
		fs = append(fs, f)
	}

	callback := reflect.ValueOf(r.callback)
	var input []reflect.Value
	callbackType := callback.Type()
	for i := 0; i < callbackType.NumIn(); i++ {
		pt := callbackType.In(i)
		for _, f := range fs {
			if reflect.TypeOf(f).AssignableTo(pt) {
				input = append(input, reflect.ValueOf(f))
				break
			}
		}
	}

	if len(input) != callbackType.NumIn() {
		panic("Can't get all input parameters")
	}

	var err error
	ret := callback.Call(input)
	errInterface := reflect.TypeOf((*error)(nil)).Elem()
	for i := len(ret) - 1; i >= 0; i-- {
		if ret[i].Type() == errInterface {
			v := ret[i].Interface()
			if v != nil {
				err = v.(error)
			}
			break
		}
	}

	return true, err
}

type featureManager struct {
	access             sync.Mutex
	features           []Feature
	featureResolutions []resolution
	running            bool
}

func newFeatureManager() *featureManager {
	return &featureManager{}
}

// RequireFeatures registers a callback, which will be called when all dependent features are registered.
// The callback must be a func(). All its parameters must be Feature.
func (s *featureManager) RequireFeatures(callback interface{}) error {
	callbackType := reflect.TypeOf(callback)
	if callbackType.Kind() != reflect.Func {
		panic("not a function")
	}

	var featureTypes []reflect.Type
	for i := 0; i < callbackType.NumIn(); i++ {
		featureTypes = append(featureTypes, reflect.PointerTo(callbackType.In(i)))
	}

	r := resolution{
		deps:     featureTypes,
		callback: callback,
	}
	if finished, err := r.resolve(s.features); finished {
		return err
	}
	s.featureResolutions = append(s.featureResolutions, r)
	return nil
}

// AddFeature registers a feature into current featureManager.
func (s *featureManager) AddFeature(feature Feature) error {
	s.features = append(s.features, feature)

	if s.running {
		if err := feature.Start(); err != nil {
			log.Printf("failed to start feature: %v", err)
		}
		return nil
	}

	if s.featureResolutions == nil {
		return nil
	}

	var pendingResolutions []resolution
	for _, r := range s.featureResolutions {
		finished, err := r.resolve(s.features)
		if finished && err != nil {
			return err
		}
		if !finished {
			pendingResolutions = append(pendingResolutions, r)
		}
	}
	if len(pendingResolutions) == 0 {
		s.featureResolutions = nil
	} else if len(pendingResolutions) < len(s.featureResolutions) {
		s.featureResolutions = pendingResolutions
	}

	return nil
}

// GetFeature returns a feature of the given type, or nil if such feature is not registered.
func (s *featureManager) GetFeature(featureType interface{}) Feature {
	return getFeature(s.features, reflect.TypeOf(featureType))
}

func (s *featureManager) UpdateFeature(feature Feature) {
	for idx, f := range s.features {
		if reflect.TypeOf(f.Type()) == reflect.TypeOf(feature.Type()) {
			if s.running {
				if err := feature.Start(); err != nil {
					log.Printf("failed to start feature: %v", err)
				}
			}
			s.features[idx] = feature
			logrus.Infof("update feature: %v", reflect.TypeOf(feature.Type()))
			f.Close()
			break
		}
	}
}

func (s *featureManager) RemoveFeature(featureType interface{}) {
	for idx, f := range s.features {
		if reflect.TypeOf(f.Type()) == reflect.TypeOf(featureType) {
			s.features[idx].Close()
			s.features = append(s.features[:idx], s.features[idx+1:]...)
			break
		}
	}
}

func ToString(v interface{}) string {
	if v == nil {
		return ""
	}

	switch value := v.(type) {
	case string:
		return value
	case *string:
		return *value
	case fmt.Stringer:
		return value.String()
	case error:
		return value.Error()
	default:
		return fmt.Sprintf("%+v", value)
	}
}

func Concat(v ...interface{}) string {
	builder := strings.Builder{}
	for _, value := range v {
		builder.WriteString(ToString(value))
	}
	return builder.String()
}

func (s *featureManager) Close() error {
	s.access.Lock()
	defer s.access.Unlock()

	s.running = false

	var errs []interface{}
	for _, f := range s.features {
		if err := f.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Wrap(errors.New(Concat(errs...)), "failed to close features")
	}

	return nil
}

func (s *featureManager) Start() error {
	s.access.Lock()
	defer s.access.Unlock()

	s.running = true
	for _, f := range s.features {
		if err := f.Start(); err != nil {
			return err
		}
	}

	return nil
}
