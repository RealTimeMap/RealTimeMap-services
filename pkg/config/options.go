package config

type Options struct {
	Paths     []string
	UseEnv    bool
	EnvPrefix string
	Required  []string
}

type Option func(*Options)

func defaultOptions() *Options {
	return &Options{
		Paths:  []string{"./config/config.yaml", "./config.yaml"},
		UseEnv: true,
	}
}

func WithPaths(paths ...string) Option {
	return func(o *Options) {
		o.Paths = paths
	}
}

func WithEnv(prefix string) Option {
	return func(o *Options) {
		o.UseEnv = true
		o.EnvPrefix = prefix
	}
}

func WithoutEnv() Option {
	return func(o *Options) {
		o.UseEnv = false
	}
}

func WithRequired(fields ...string) Option {
	return func(o *Options) {
		o.Required = fields
	}
}
