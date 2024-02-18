package streaminterceptor

type Attributes map[interface{}]interface{}

// Get returns the attribute associated with key.
func (a Attributes) Get(key interface{}) interface{} {
	return a[key]
}

// Set sets the attribute associated with key to the given value.
func (a Attributes) Set(key interface{}, val interface{}) {
	a[key] = val
}
