package langfuse

// options holds configuration for GetPrompt calls.
type options struct {
	label   string
	version int
}

// Option configures a GetPrompt call.
type Option func(*options)

// defaultOptions returns the default options.
func defaultOptions() *options {
	return &options{
		label: "production",
	}
}

// WithLabel sets the label to filter prompts by.
// Default is "production".
func WithLabel(label string) Option {
	return func(o *options) {
		o.label = label
	}
}

// WithVersion sets a specific version to fetch.
// If set, label is ignored.
func WithVersion(version int) Option {
	return func(o *options) {
		o.version = version
	}
}

// WithoutLabel removes the default label filter.
func WithoutLabel() Option {
	return func(o *options) {
		o.label = ""
	}
}
