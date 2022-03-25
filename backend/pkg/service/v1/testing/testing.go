package testing

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	goTesting "testing"

	"github.com/DagDigg/unpaper/backend/helpers"
	v1API "github.com/DagDigg/unpaper/backend/pkg/api/v1"
	v1 "github.com/DagDigg/unpaper/backend/pkg/service/v1"
	"github.com/DagDigg/unpaper/backend/users"
	"github.com/DagDigg/unpaper/core/config"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	k8sAPIv1 "k8s.io/api/core/v1"
	k8sMetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

// WrappedServer is an UnpaperServiceServer wrapper for testing
// purposes.
// It also exports its config and context
type WrappedServer struct {
	Server v1.Server
	Cfg    *config.Config
	Ctx    context.Context
}

// GetWrappedServer starts a docker container running postgres.
// It then creates an UnpaperServiceServer with the docker postgres connection string
// and returns a wrappedServer, which contains the newly created server
func GetWrappedServer(t *goTesting.T) *WrappedServer {
	ctx := context.Background()
	cfg := InitConfig()
	dbConnURL := helpers.StartDatabase(t, cfg.GetDBConnURL())
	rdbConnURL := helpers.StartRedisDB(t, cfg.GetRDBConnURL())
	cfg.Environment = "dev"
	cfg.DatastoreDBHost = dbConnURL.Host
	cfg.DatastoreDBPort = dbConnURL.Port()
	cfg.RedisHost = rdbConnURL.Host
	cfg.RedisPort = rdbConnURL.Port()
	server, err := v1.NewUnpaperServiceServer(cfg)
	if err != nil {
		t.Errorf("error creating service server: '%v'", err)
	}
	return &WrappedServer{
		server,
		cfg,
		ctx,
	}
}

// InitConfig creates a fake k8s client, feeded with env as k8s secrets,
// and returns the full config
func InitConfig() *config.Config {
	fakeClient := testclient.NewSimpleClientset()
	createK8sSecret(fakeClient)
	return config.Get(config.Params{
		K8sClientSet: fakeClient,
	})
}

// AddUser inserts a mocked user into db
func (s *WrappedServer) AddUser(p users.CreateUserParams) (*v1API.User, error) {
	// Insert fresh user
	dir := users.NewDirectory(s.Server.GetDB())

	// Hash password, as it do not need to be plain
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(p.Password.String), 8)
	if err != nil {
		return nil, fmt.Errorf("failed to hash mocked user password: '%v'", err)
	}
	p.Password = sql.NullString{String: string(hashedPassword), Valid: true}

	user, err := dir.CreateUser(s.Ctx, p)
	if err != nil {
		return nil, fmt.Errorf("error inserting user: '%v'", err)
	}

	return user, nil
}

// DeleteUser deletes the mockedPGUser from db
func (s *WrappedServer) DeleteUser(id string) error {
	dir := users.NewDirectory(s.Server.GetDB())

	return dir.DeleteUser(s.Ctx, id)
}

// AssertUser compares the protobuf passed user, to the mock one.
// It needs to be called if the user has been previously added with 'AddUser'
// since it strictly compares to the mocked user
func AssertUser(t *goTesting.T, pbUser *v1API.User, p users.CreateUserParams) {
	// Asserts properties of inserted user and returned user
	if pbUser.Id != p.ID {
		t.Errorf("user property 'id' mismatch. got: '%v', want: '%v'", pbUser.Id, p.ID)
	}
	if pbUser.Email != p.Email {
		t.Errorf("user property 'email' mismatch. got: '%v', want: '%v'", pbUser.Email, p.Email)
	}
	if pbUser.FamilyName != p.FamilyName.String {
		t.Errorf("user property 'familyName' mismatch. got: '%v', want: '%v'", pbUser.FamilyName, p.FamilyName.String)
	}
	if pbUser.GivenName != p.GivenName.String {
		t.Errorf("user property 'givenName' mismatch. got: '%v', want: '%v'", pbUser.GivenName, p.GivenName)
	}
}

// GetRandomPGUserParams returns a random postgres create user params
func GetRandomPGUserParams() users.CreateUserParams {
	return users.CreateUserParams{
		ID:         uuid.NewString(),
		Email:      uuid.NewString() + "@gmail.com",
		Username:   sql.NullString{String: uuid.NewString(), Valid: true},
		GivenName:  sql.NullString{String: uuid.NewString(), Valid: true},
		FamilyName: sql.NullString{String: uuid.NewString(), Valid: true},
		Password:   sql.NullString{String: uuid.NewString(), Valid: true},
	}
}

func createK8sSecret(k kubernetes.Interface) {
	secrets := &k8sAPIv1.Secret{
		Data:       getSecretsDataFromENV(),
		ObjectMeta: k8sMetav1.ObjectMeta{Name: "secrets"},
	}
	_, err := k.CoreV1().Secrets(k8sAPIv1.NamespaceDefault).Create(context.Background(), secrets, k8sMetav1.CreateOptions{})
	if err != nil {
		log.Fatal(err)
	}
}

func getSecretsDataFromENV() map[string][]byte {
	return map[string][]byte{
		"postgres_user":         envToB64("POSTGRES_USER"),
		"postgres_password":     envToB64("POSTGRES_PASSWORD"),
		"google_client_id":      envToB64("GOOGLE_CLIENT_ID"),
		"google_client_secret":  envToB64("GOOGLE_CLIENT_SECRET"),
		"unpaper_client_secret": envToB64("UNPAPER_CLIENT_SECRET"),
		"smtp_user":             envToB64("SMTP_USER"),
		"smtp_pass":             envToB64("SMTP_PASS"),
		"stripe_api_key":        envToB64("STRIPE_API_KEY"),
		"redis_pass":            envToB64("REDIS_PASS"),
		"client_domain":         envToB64("CLIENT_DOMAIN"),
	}
}

func envToB64(k string) []byte {
	return []byte(base64.StdEncoding.EncodeToString([]byte(os.Getenv(k))))
}
