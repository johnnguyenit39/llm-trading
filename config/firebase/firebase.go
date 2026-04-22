// Package firecfg loads Firebase Admin credentials from the environment
// (base64 JSON) and builds a Firebase App. Use for FCM, Auth, or other
// Firebase Admin APIs.
package firecfg

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// EnvServiceAccountBase64 is the env var holding the service account
// JSON encoded as standard base64 (as in Google Cloud / CI secrets).
const EnvServiceAccountBase64 = "SERVICE_ACCOUNT_FIREBASE_BASE_64"

// App is the process-wide Firebase app after the composition root sets it
// from the env (e.g. main). Nil when EnvServiceAccountBase64 is unset or
// not yet initialized.
var App *firebase.App

// CredentialsJSONFromEnv returns decoded service account JSON if
// EnvServiceAccountBase64 is set. Empty env → nil, nil. Invalid base64
// or empty payload after decode → error.
func CredentialsJSONFromEnv() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(EnvServiceAccountBase64))
	if raw == "" {
		return nil, nil
	}
	raw = strings.Trim(raw, `"'`)
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", EnvServiceAccountBase64, err)
	}
	if len(decoded) == 0 {
		return nil, fmt.Errorf("%s: decoded to empty bytes", EnvServiceAccountBase64)
	}
	return decoded, nil
}

// NewAppFromEnv builds a default Firebase app from credentials in
// EnvServiceAccountBase64. If the env var is unset, returns nil, nil
// (Firebase is optional for processes that do not use it).
func NewAppFromEnv(ctx context.Context) (*firebase.App, error) {
	jsonBytes, err := CredentialsJSONFromEnv()
	if err != nil {
		return nil, err
	}
	if jsonBytes == nil {
		return nil, nil
	}
	return firebase.NewApp(ctx, nil, option.WithCredentialsJSON(jsonBytes))
}

// MessagingClient returns the FCM client for app, or nil if app is nil.
func MessagingClient(ctx context.Context, app *firebase.App) (*messaging.Client, error) {
	if app == nil {
		return nil, nil
	}
	return app.Messaging(ctx)
}

// Messaging returns the FCM client for [App], or nil if [App] is nil.
func Messaging(ctx context.Context) (*messaging.Client, error) {
	return MessagingClient(ctx, App)
}

// FirestoreClient returns a Firestore client bound to app's project, or
// nil if app is nil. Caller owns the client and must Close() it on
// shutdown.
func FirestoreClient(ctx context.Context, app *firebase.App) (*firestore.Client, error) {
	if app == nil {
		return nil, nil
	}
	return app.Firestore(ctx)
}

// Firestore returns a Firestore client bound to [App], or nil if [App]
// is nil.
func Firestore(ctx context.Context) (*firestore.Client, error) {
	return FirestoreClient(ctx, App)
}
