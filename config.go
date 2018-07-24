package mesosauth

import (
	"context"
	"time"

	"github.com/hashicorp/vault/logical"
	"github.com/hashicorp/vault/logical/framework"
)

const defaultTTL = 10 * time.Minute

// pathConfig returns the "config" path struct. It is a function rather than a
// method because we never call it once the backend struct is built and we
// don't want name collisions with any request handler methods.
func pathConfig(b *mesosBackend) *framework.Path {
	return &framework.Path{
		Pattern: "config",
		Fields: map[string]*framework.FieldSchema{
			"base-url": {
				Type:        framework.TypeString,
				Description: "Mesos API base URL.",
			},
			"ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Duration after which authentication will be expired",
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.CreateOperation: b.pathConfigWrite,
			logical.UpdateOperation: b.pathConfigWrite,
			logical.ReadOperation:   b.pathConfigRead,
		},
	}
}

// config is used to store plugin configuration.
type config struct {
	BaseURL string
	TTL     time.Duration
}

// configDefault returns a new config containing default settings.
func configDefault() *config {
	return &config{
		TTL: defaultTTL,
	}
}

// pathConfigWrite is the "config" create/update request handler.
func (b *mesosBackend) pathConfigWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	rh := requestHelper{ctx: ctx, storage: req.Storage}

	// TODO: Decide if we want to allow invalid configs to be overwritten.
	cfg, err := rh.getConfigOrNil()
	if err != nil {
		return nil, err
	}

	// If we don't already have a stored config, we're creating a new one and
	// must thus start with defaults.
	if cfg == nil {
		cfg = configDefault()
	}

	if baseURL, ok := d.GetOk("base-url"); ok {
		cfg.BaseURL = baseURL.(string)
	}

	if ttl, ok := d.GetOk("ttl"); ok {
		cfg.TTL = time.Duration(ttl.(int)) * time.Second
	}

	if cfg.BaseURL == "" {
		return logical.ErrorResponse("base-url not configured"), nil
	}

	err = rh.store("config", cfg)
	return &logical.Response{}, err
}

// pathConfigRead is the "config" read request handler.
func (b *mesosBackend) pathConfigRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	rh := requestHelper{ctx: ctx, storage: req.Storage}

	cfg, err := rh.getConfigOrNil()
	if cfg == nil || err != nil {
		return nil, err
	}

	resp := &logical.Response{
		Data: jsonobj{
			"base-url": cfg.BaseURL,
			"ttl":      cfg.TTL.String(),
		},
	}
	return resp, nil
}
