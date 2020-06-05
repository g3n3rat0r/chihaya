package dcd

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/chihaya/chihaya/bittorrent"
	"github.com/chihaya/chihaya/middleware"
	"github.com/chihaya/chihaya/pkg/sql"
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
	DB            []string `yaml:"database"`
	MySQLUser     string   `yaml:"mysqluser"`
	MySQLPassword string   `yaml:"mysqlpass"`
	MySQLDatabase string   `yaml:"mysqldb"`
}

type hook struct {
	db       map[string]struct{}
	passkeys []string
}

// NewHook returns an instance of the client approval middleware.
func NewHook(cfg Config) (middleware.Hook, error) {
	h := &hook{
		db: make(map[string]struct{}),
	}
	fmt.Println(cfg)
	mDB, err := sql.OpenSql(cfg.MySQLUser, cfg.MySQLPassword, cfg.MySQLDatabase)
	if err != nil {
		return h, err
	}

	keys, err := mDB.GetEnabledPassKeys()
	if err != nil {
		return h, err
	}

	h.passkeys = keys

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
	for _, v := range h.passkeys {
		if v == passkey {
			return ctx, nil
		}
	}
	return nil, fmt.Errorf("Invalid passkey")
}

func (h *hook) HandleScrape(ctx context.Context, req *bittorrent.ScrapeRequest, resp *bittorrent.ScrapeResponse) (context.Context, error) {
	// Scrapes don't require any protection.
	return ctx, nil
}
