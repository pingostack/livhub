package streaminterceptor

type Registry struct {
	factories []StreamFactory
}

func (r *Registry) Add(f StreamFactory) {
	r.factories = append(r.factories, f)
}

func (r *Registry) Build(id string) (Interceptor, error) {
	if len(r.factories) == 0 {
		return &NoOp{}, nil
	}

	interceptors := []Interceptor{}
	for _, f := range r.factories {
		i, err := f.NewInterceptor(id)
		if err != nil {
			return nil, err
		}

		interceptors = append(interceptors, i)
	}

	return NewChain(interceptors), nil
}
