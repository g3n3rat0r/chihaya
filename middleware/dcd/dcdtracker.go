package dcd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"gopkg.in/yaml.v2"
)

// Name is the name by which this middleware is registered with Chihaya.
const Name = "desi.cd tracker"

func init() {
	middleware.RegisterDriver(Name, driver{})
}

var _ middleware.Driver = driver{}

type driver struct{}

func (d driver) NewHook(optionBytes []byte) (middleware.Hook, error) {
	var cfg Config
	err := yaml.Unmarshal(optionBytes, &cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid options for middleware %s: %s", Name, err)
	}

	return NewHook(cfg)
}

type Config struct {
	DB []string `yaml:"database"`
}

type hook struct {
	db map[string]struct{}
}

// NewHook returns an instance of the client approval middleware.
func NewHook(cfg Config) (middleware.Hook, error) {
	h := &hook{
		db: make(map[string]struct{}),
	}

	return h, nil
}

func (h *hook) HandleAnnounce(ctx context.Context, req *bittorrent.AnnounceRequest, resp *bittorrent.AnnounceResponse) (context.Context, error) {
	log.Println("Will handle desi.cd announce")
	// Extract passkey from query path
	path := req.Params.RawPath()
	f := func(c rune) bool {
		return c == '/'
	}
	passkey := strings.FieldsFunc(path, f)[0]
	fmt.Println("passkey ", passkey)
	if !(len(passkey) > 10) {
		return nil, fmt.Errorf("Invalid passkey")
	}
	return ctx, nil
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes don't require any protection.
	return ctx, nil
}
