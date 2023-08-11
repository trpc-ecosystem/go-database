package cos

import "net/url"

// options are options of cos request.
type options struct {
	headers map[string]string
	params  map[string]string
}

// GetParamString concatenates the request parameter string.
func (o options) GetParamString() string {
	if len(o.params) == 0 {
		return ""
	}

	val := url.Values{}
	for k, v := range o.params {
		val.Set(k, v)
	}

	return val.Encode()
}

// GetHeader returns the request header.
func (o options) GetHeader() map[string]string {
	return o.headers
}

// GetParam returns the request params.
func (o options) GetParam() map[string]string {
	return o.params
}

// NewOptions returns cos options.
func NewOptions() *options {
	o := &options{
		params:  make(map[string]string),
		headers: make(map[string]string),
	}

	return o
}

// Option sets cos options.
type Option func(o *options)

// WithPrefix returns an Option that sets object prefix.
func WithPrefix(prefix string) Option {
	return func(opts *options) {
		opts.params["prefix"] = prefix
	}
}

// WithDelimiter returns an Option that sets delimiter.
func WithDelimiter(delimiter string) Option {
	return func(opts *options) {
		opts.params["delimiter"] = delimiter
	}
}

// WithKeyMaker returns an Option that sets the starting key for the result.
func WithKeyMaker(keyMaker string) Option {
	return func(opts *options) {
		opts.params["key-maker"] = keyMaker
	}
}

// WithVersionIDMarker returns an Option that sets the starting version for the result,
// It's empty by default and need to be used with key-maker to take effect.
func WithVersionIDMarker(versionIDMarker string) Option {
	return func(opts *options) {
		opts.params["version-id-marker"] = versionIDMarker
	}
}

// WithMaxKeys return an Option that sets the maximum number of results to return,
// the actual number of results may be less than this number.
func WithMaxKeys(maxKeys string) Option {
	return func(opts *options) {
		opts.params["max-keys"] = maxKeys
	}
}

// WithMarker return an Option that sets the starting key for the result.
func WithMarker(marker string) Option {
	return func(opts *options) {
		opts.params["marker"] = marker
	}
}

// WithVersionID returns an Option that sets the version id,
// it's used to download files of a specified version from multi-versioned buckets.
func WithVersionID(versionID string) Option {
	return func(opts *options) {
		opts.params["versionid"] = versionID
	}
}

// WithCustomParam returns an Option that sets the custom parameters.
func WithCustomParam(key, val string) Option {
	return func(opts *options) {
		opts.params[key] = val
	}
}

// WithCustomHeader returns an Option that sets the custom header.
func WithCustomHeader(key, val string) Option {
	return func(opts *options) {
		opts.headers[key] = val
	}
}
