package config

import (
	"context"
	"flag"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/joho/godotenv"
)

// Config is configuration for Server
type Config struct {
	// API server start parameters section
	// APIPort is TCP port to listen for gRPC calls
	// and HTTP ones
	APIPort string

	// Port where the external authorization server will listen to
	AuthServerPort string

	// DB Datastore parameters section
	// DatastoreDBHost is host of database
	DatastoreDBHost string
	// DatastoreDBUser is username to connect to database
	DatastoreDBUser string
	// DatastoreDBPassword password to connect to database
	DatastoreDBPassword string
	// DatastoreDBPort port to connect to
	DatastoreDBPort string
	// DatastoreDBName name of the database
	DatastoreDBName string

	// Log parameters section
	// LogLevel is global log level: Debug(-1), Info(0), Warn(1), Error(2), DPanic(3), Panic(4), Fatal(5)
	LogLevel int
	// LogTimeFormat is print time format for logger e.g. 2006-01-02T15:04:05Z07:00
	LogTimeFormat string

	// Client IDs and secrets
	GoogleClientID     string
	GoogleClientSecret string

	UnpaperClientSecret string

	// SMTP transactional emails
	SMTPDomain string
	SMTPPort   string
	SMTPUser   string
	SMTPPass   string

	// Environment
	Environment string

	// Stripe
	StripeAPIKey               string
	PriceIDFree                string
	PriceIDPlusMonthly         string
	PriceIDPlusYearly          string
	ProductIDRoomSubscriptions string

	// Redis
	RedisHost string
	RedisPort string
	RedisPass string

	// Payment
	ApplicationFeePercent string

	// Development and testing
	EnableStripe bool

	// Client domain refers to the trusted client domain
	ClientDomain string
}

var (
	cfg *Config     = &Config{}
	mu  *sync.Mutex = &sync.Mutex{}
)

func init() {
	// env variables are not available locally,
	// so they need to be loaded from where they're declared:
	// outside of the module, at the root of the project.
	_, f, _, _ := runtime.Caller(0)
	currDir := filepath.Dir(f)
	p := filepath.Join(currDir, "../../.env")
	godotenv.Load(p)

	if os.Getenv("API_PORT") == "" {
		log.Fatal("NO ENV")
	}

	// parse flags, or use .env as default
	flag.StringVar(&cfg.APIPort, "api-port", os.Getenv("API_PORT"), "API port to bind")
	flag.StringVar(&cfg.AuthServerPort, "auth-port", os.Getenv("AUTH_SERVER_PORT"), "Authorization server port")

	flag.StringVar(&cfg.DatastoreDBHost, "db-host", os.Getenv("POSTGRES_HOST"), "Database host")

	flag.StringVar(&cfg.DatastoreDBPort, "db-port", os.Getenv("POSTGRES_PORT"), "Database port")
	flag.StringVar(&cfg.DatastoreDBName, "db-name", os.Getenv("POSTGRES_DB"), "Database name")

	flag.IntVar(&cfg.LogLevel, "log-level", -1, "global log level")
	flag.StringVar(&cfg.LogTimeFormat, "log-time-format", os.Getenv("LOG_TIME_FORMAT"), "logging time formatting")

	flag.StringVar(&cfg.UnpaperClientSecret, "unpaper-client-secret", os.Getenv("UNPAPER_CLIENT_SECRET"), "Unpaper client secret")

	flag.StringVar(&cfg.SMTPDomain, "smtp-domain", os.Getenv("SMTP_DOMAIN"), "smtp domain host")
	flag.StringVar(&cfg.SMTPPort, "smtp-port", os.Getenv("SMTP_PORT"), "smtp domain port")

	flag.StringVar(&cfg.Environment, "environment", os.Getenv("ENVIRONMENT"), "environment of the application: 'prod' or 'dev'")

	flag.StringVar(&cfg.PriceIDFree, "price-id-free", os.Getenv("PRICE_ID_FREE"), "Stripe price ID for the free version")
	flag.StringVar(&cfg.PriceIDPlusMonthly, "price-id-plus-monthly", os.Getenv("PRICE_ID_PLUS_MONTHLY"), "Stripe price ID for the monthly plus version")
	flag.StringVar(&cfg.PriceIDPlusYearly, "price-id-plus-yearly", os.Getenv("PRICE_ID_PLUS_YEARLY"), "Stripe price ID for the yearly plus version")
	flag.StringVar(&cfg.ProductIDRoomSubscriptions, "product-id-room-subscriptions", os.Getenv("PRODUCT_ID_ROOM_SUBSCRIPTIONS"), "Stripe Product ID for the room subscriptions")

	flag.StringVar(&cfg.RedisHost, "redis-host", os.Getenv("REDIS_HOST"), "Redis host")
	flag.StringVar(&cfg.RedisPort, "redis-port", os.Getenv("REDIS_PORT"), "Port for connecting to redis")

	flag.StringVar(&cfg.ApplicationFeePercent, "app-fee-perc", os.Getenv("APPLICATION_FEE_PERCENT"), "Fee percent amount to be taken from donations")

	flag.BoolVar(&cfg.EnableStripe, "enable-stripe", isStringTrue(os.Getenv("ENABLE_STRIPE")), "Set this flag to true ")
}

func isStringTrue(s string) bool {
	return s == "true"
}

// Params used for getting configuration
type Params struct {
	K8sClientSet kubernetes.Interface
}

// Get returns the Config structure with variables picked from .env
func Get(p Params) *Config {
	mu.Lock()
	defer mu.Unlock()
	flag.Parse()
	addSecretsToCfg(p.K8sClientSet)
	return cfg
}

// GetDBConnURL returns postgres connection string as *url.URL
// Default data is retrieved from env variables
func (c *Config) GetDBConnURL() *url.URL {
	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DatastoreDBUser, c.DatastoreDBPassword),
		Path:   "unpaper",
	}
	q := pgURL.Query()
	q.Add("sslmode", "disable")
	pgURL.RawQuery = q.Encode()
	pgURL.Host = c.DatastoreDBHost

	return pgURL
}

// GetRDBConnURL returns the redis connection *url.URL
func (c *Config) GetRDBConnURL() *url.URL {
	rdbURL := &url.URL{
		Scheme: "redis",
		User:   url.UserPassword("", c.RedisPass),
		Path:   "0",
		Host:   c.RedisHost,
	}
	return rdbURL
}

// CtxKey is a type used for context key to avoid collisions
// with other packages
type CtxKey string

// addSecretsToCfg retrieves the secrets k8s object in the cluster and applies them to cfg
func addSecretsToCfg(k kubernetes.Interface) {
	s, err := k.CoreV1().Secrets(apiv1.NamespaceDefault).Get(context.Background(), "secrets", v1.GetOptions{})
	if err != nil {
		log.Fatalf("error getting secrets: %v", err)
	}

	cfg.DatastoreDBUser = string(s.Data["postgres_user"])
	cfg.DatastoreDBPassword = string(s.Data["postgres_password"])
	cfg.GoogleClientID = string(s.Data["google_client_id"])
	cfg.GoogleClientSecret = string(s.Data["google_client_secret"])
	cfg.SMTPUser = string(s.Data["smtp_user"])
	cfg.SMTPPass = string(s.Data["smtp_pass"])
	cfg.StripeAPIKey = string(s.Data["stripe_api_key"])
	cfg.ClientDomain = string(s.Data["client_domain"])
}
