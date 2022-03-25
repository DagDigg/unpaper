package secrets

// Secrets contains client IDs and client secrets
// used to authorize client with google or to sign/validate
// custom tokens
type Secrets struct {
	GoogleClientID      string
	GoogleClientSecret  string
	UnpaperClientSecret string
}
