package plugins

import context "github.com/arthurlch/goryu/goryuctx"

type Plugin interface {
	// Name returns the plugin name
	Name() string
	Build() context.Middleware
	Validate() error
}

type Builder interface {
	Plugin
	Reset() Builder
	Clone() Builder
}

type Registry struct {
	plugins map[string]func() Builder
}

var DefaultRegistry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]func() Builder),
	}
}

func (r *Registry) Register(name string, factory func() Builder) {
	r.plugins[name] = factory
}

func (r *Registry) Get(name string) (Builder, bool) {
	factory, exists := r.plugins[name]
	if !exists {
		return nil, false
	}
	return factory(), true
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	return names
}

func Register(name string, factory func() Builder) {
	DefaultRegistry.Register(name, factory)
}

func Get(name string) (Builder, bool) {
	return DefaultRegistry.Get(name)
}

func List() []string {
	return DefaultRegistry.List()
}

type BaseBuilder struct {
	name     string
	errors   []error
	metadata map[string]interface{}
}

func NewBaseBuilder(name string) *BaseBuilder {
	return &BaseBuilder{
		name:     name,
		errors:   make([]error, 0),
		metadata: make(map[string]interface{}),
	}
}

func (b *BaseBuilder) Name() string {
	return b.name
}

func (b *BaseBuilder) AddError(err error) {
	b.errors = append(b.errors, err)
}

func (b *BaseBuilder) Validate() error {
	if len(b.errors) == 0 {
		return nil
	}
	return b.errors[0]
}

func (b *BaseBuilder) SetMetadata(key string, value interface{}) {
	b.metadata[key] = value
}

func (b *BaseBuilder) GetMetadata(key string) (interface{}, bool) {
	value, exists := b.metadata[key]
	return value, exists
}

func (b *BaseBuilder) ClearErrors() {
	b.errors = b.errors[:0]
}