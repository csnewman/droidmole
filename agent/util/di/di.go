package di

import (
	"errors"
	"go.uber.org/dig"
	"reflect"
)

type config struct {
	providers []provider
}

type provider struct {
	constructor interface{}
	opts        []dig.ProvideOption
}

type Option interface {
	apply(*config)
}

type Container struct {
	dc *dig.Container
}

func New(opts ...Option) (*Container, error) {
	// Extract opts
	conf := config{}
	for _, opt := range opts {
		opt.apply(&conf)
	}

	// Build
	dc := dig.New(dig.DeferAcyclicVerification())

	for _, p := range conf.providers {
		err := dc.Provide(p.constructor, p.opts...)
		if err != nil {
			return nil, err
		}
	}

	return &Container{dc: dc}, nil
}

func (c *Container) Get(target any) error {
	// TODO: Replace with generics once supported
	rt := reflect.TypeOf(target)
	if rt.Kind() != reflect.Ptr {
		return errors.New("expected pointer")
	}

	args := []reflect.Type{
		rt.Elem(),
	}
	fnType := reflect.FuncOf(args, nil, false)
	fn := reflect.MakeFunc(fnType, func(args []reflect.Value) []reflect.Value {
		reflect.ValueOf(target).Elem().Set(args[0])
		return nil
	})
	return c.dc.Invoke(fn.Interface())
}

type providerOpt struct {
	p provider
}

func (po providerOpt) apply(c *config) {
	c.providers = append(c.providers, po.p)
}

func Provider(constructor interface{}, opts ...dig.ProvideOption) Option {
	return &providerOpt{
		p: provider{
			constructor: constructor,
			opts:        opts,
		},
	}
}
