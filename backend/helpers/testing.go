package helpers

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-migrate/migrate/v4"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	// golang migrate postgres driver
	_ "github.com/golang-migrate/migrate/v4/database/postgres"

	// golang migrate file driver
	_ "github.com/golang-migrate/migrate/v4/source/file"

	// pgx postgres driver
	_ "github.com/jackc/pgx/v4/stdlib"
)

// StartDatabase creates a docker container running postgres,
// performs UP migration on startup, DOWN migration on cleanup, purging the container.
// It then returns the postgres connection URL to the pg container
func StartDatabase(tb testing.TB, pgURL *url.URL) *url.URL {
	tb.Helper()

	pool, err := dockertest.NewPool("")
	if err != nil {
		tb.Fatalf("error starting docker: %v", err)
	}

	pw, _ := pgURL.User.Password()
	env := []string{
		"POSTGRES_USER=" + pgURL.User.Username(),
		"POSTGRES_PASSWORD=" + pw,
		"POSTGRES_DB=" + pgURL.Path,
	}
	resource, err := pool.Run("postgres", "13-alpine", env)
	if err != nil {
		tb.Fatalf("error starting postgres container: %v", err)
	}

	tb.Cleanup(func() {
		// Remove container and linked volumes
		err = pool.Purge(resource)
		if err != nil {
			tb.Fatalf("error purging container: %v", err)
		}
	})

	// Set the IPAddress of the container as postgres host
	pgURL.Host = resource.Container.NetworkSettings.IPAddress

	// Docker layer network is different on Mac
	if runtime.GOOS == "darwin" {
		pgURL.Host = net.JoinHostPort(resource.GetBoundIP("5432/tcp"), resource.GetPort("5432/tcp"))
	}

	logWaiter, err := pool.Client.AttachToContainerNonBlocking(docker.AttachToContainerOptions{
		Container:    resource.Container.ID,
		OutputStream: log.Writer(),
		ErrorStream:  log.Writer(),
		Stderr:       true,
		Stdout:       true,
		Stream:       true,
	})
	if err != nil {
		tb.Fatalf("error connecting to postgres container log output: %v", err)
	}

	// Cleanup after test
	tb.Cleanup(func() {
		err = logWaiter.Close()
		if err != nil {
			tb.Fatalf("error closing container log: %v", err)
		}
		err = logWaiter.Wait()
		if err != nil {
			tb.Fatalf("error waiting for container log to close: %v", err)
		}
	})

	pool.MaxWait = 10 * time.Second
	err = pool.Retry(func() (err error) {
		db, err := sql.Open("pgx", pgURL.String())
		if err != nil {
			return err
		}
		defer func() {
			cerr := db.Close()
			if err == nil {
				err = cerr
			}
		}()

		return db.Ping()
	})

	// Run database migrations
	m, err := migrate.New("file://"+migrationsFilePath(), pgURL.String())
	if err != nil {
		tb.Fatalf("err creating migration: %v", err)
	}

	// Migrate UP
	if err := m.Up(); err != nil {
		tb.Fatalf("error migrating up: %v", err)
	}

	// Return the postgres URL, which is exactly
	// the one passed as parameter, but with the host
	// set with the docker one
	return pgURL
}

// mifrationsFilePath returns the path where
// migrations are declared
func migrationsFilePath() string {
	curr := currDir()
	return filepath.Join(curr, "../../core/db/migrations")
}

// currDir returns the parent directory of
// the file invocating this function
func currDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Dir(f)
}

// StartRedisDB starts a Redis database on a docker pool
func StartRedisDB(tb testing.TB, rdbURL *url.URL) *url.URL {
	var db *redis.Client
	var err error
	pool, err := dockertest.NewPool("")
	if err != nil {
		tb.Fatalf("Could not connect to docker: %s", err)
	}
	pw, ok := rdbURL.User.Password()
	if !ok {
		tb.Fatalf("missing redis password in connection string")
	}
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "redis",
		Tag:        "alpine",
		Cmd:        []string{"redis-server", "--requirepass", pw},
	})
	if err != nil {
		tb.Fatalf("Could not start resource: %s", err)
	}

	// Set the IPAddress of the container as postgres host
	rdbURL.Host = resource.Container.NetworkSettings.IPAddress

	// Docker layer network is different on Mac
	if runtime.GOOS == "darwin" {
		rdbURL.Host = net.JoinHostPort(resource.GetBoundIP("6379/tcp"), resource.GetPort("6379/tcp"))
	}

	if err = pool.Retry(func() error {
		opt, err := redis.ParseURL(rdbURL.String())
		if err != nil {
			tb.Fatalf("error parsing redis url: %v", err)
		}

		db = redis.NewClient(opt)
		return db.Ping(context.Background()).Err()
	}); err != nil {
		tb.Fatalf("Could not connect to docker: %s", err)
	}

	tb.Cleanup(func() {
		// Remove container and linked volumes
		if err = pool.Purge(resource); err != nil {
			tb.Fatalf("Could not purge resource: %s", err)
		}
	})

	return rdbURL
}

// GetRDBInstance returns redis instance given its connection URL
func GetRDBInstance(t *testing.T, connURL *url.URL) *redis.Client {
	opt, err := redis.ParseURL(connURL.String())
	if err != nil {
		t.Fatal(err)
	}

	return redis.NewClient(opt)
}

// SameStringSlice returns whether two slices contains
// the exact same items. (order does not matter)
func SameStringSlice(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	// create a map of string -> int
	diff := make(map[string]int, len(x))
	for _, _x := range x {
		// 0 value for int is 0, so just increment a counter for the string
		diff[_x]++
	}
	for _, _y := range y {
		// If the string _y is not in diff bail out early
		if _, ok := diff[_y]; !ok {
			return false
		}
		diff[_y]--
		if diff[_y] == 0 {
			delete(diff, _y)
		}
	}

	return len(diff) == 0
}
