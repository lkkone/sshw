package configserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db    *sql.DB
	vault *vault
}

type ProfileState struct {
	Profile          string       `json:"profile"`
	Document         Document     `json:"document"`
	YAML             string       `json:"yaml"`
	DraftHash        string       `json:"draftHash"`
	PublishedVersion int          `json:"publishedVersion"`
	PublishedHash    string       `json:"publishedHash"`
	UpdatedAt        time.Time    `json:"updatedAt"`
	History          []Version    `json:"history"`
	Tokens           []TokenEntry `json:"tokens"`
}

type PublishedConfig struct {
	Profile string
	Version int
	YAML    []byte
	SHA256  string
	Note    string
	Created time.Time
}

type Version struct {
	Version   int       `json:"version"`
	SHA256    string    `json:"sha256"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"createdAt"`
}

type TokenEntry struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Profile   string     `json:"profile"`
	CreatedAt time.Time  `json:"createdAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
}

func OpenStore(path, masterKey string) (*Store, error) {
	v, err := newVault(masterKey)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dsn := (&url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}).String() +
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	store := &Store{db: db, vault: v}
	if err := store.initialize(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	if err := os.Chmod(path, 0600); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) initialize(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS profiles (
			name TEXT PRIMARY KEY,
			draft_document BLOB NOT NULL,
			draft_yaml BLOB NOT NULL,
			draft_hash TEXT NOT NULL,
			published_version INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			profile TEXT NOT NULL,
			version INTEGER NOT NULL,
			config_yaml BLOB NOT NULL,
			sha256 TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY(profile) REFERENCES profiles(name) ON DELETE CASCADE,
			UNIQUE(profile, version)
		)`,
		`CREATE TABLE IF NOT EXISTS sync_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			profile TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			created_at TEXT NOT NULL,
			revoked_at TEXT,
			FOREIGN KEY(profile) REFERENCES profiles(name) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_profile_version
			ON versions(profile, version DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_sync_tokens_profile_active
			ON sync_tokens(profile, revoked_at)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	_, err := s.db.ExecContext(ctx, "PRAGMA optimize")
	return err
}

func (s *Store) EnsureProfile(ctx context.Context, profile string) error {
	document, _ := EncodeDocument(EmptyDocument())
	yamlData := []byte("[]\n")
	encryptedDocument, err := s.vault.encrypt(document)
	if err != nil {
		return err
	}
	encryptedYAML, err := s.vault.encrypt(yamlData)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(yamlData)
	_, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO profiles
		(name, draft_document, draft_yaml, draft_hash, published_version, updated_at)
		VALUES (?, ?, ?, ?, 0, ?)`,
		profile, encryptedDocument, encryptedYAML, hex.EncodeToString(sum[:]), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) SaveDraft(ctx context.Context, profile string, document Document, yamlData []byte) error {
	documentData, err := EncodeDocument(document)
	if err != nil {
		return err
	}
	encryptedDocument, err := s.vault.encrypt(documentData)
	if err != nil {
		return err
	}
	encryptedYAML, err := s.vault.encrypt(yamlData)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(yamlData)
	result, err := s.db.ExecContext(ctx, `UPDATE profiles
		SET draft_document = ?, draft_yaml = ?, draft_hash = ?, updated_at = ?
		WHERE name = ?`,
		encryptedDocument, encryptedYAML, hex.EncodeToString(sum[:]), time.Now().UTC().Format(time.RFC3339Nano), profile)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("profile %q does not exist", profile)
	}
	return nil
}

func (s *Store) State(ctx context.Context, profile string) (ProfileState, error) {
	var encryptedDocument, encryptedYAML []byte
	var state ProfileState
	var updated string
	err := s.db.QueryRowContext(ctx, `SELECT name, draft_document, draft_yaml, draft_hash,
		published_version, updated_at FROM profiles WHERE name = ?`, profile).
		Scan(&state.Profile, &encryptedDocument, &encryptedYAML, &state.DraftHash, &state.PublishedVersion, &updated)
	if err != nil {
		return ProfileState{}, err
	}
	documentData, err := s.vault.decrypt(encryptedDocument)
	if err != nil {
		return ProfileState{}, err
	}
	state.Document, err = DecodeDocument(documentData)
	if err != nil {
		return ProfileState{}, err
	}
	yamlData, err := s.vault.decrypt(encryptedYAML)
	if err != nil {
		return ProfileState{}, err
	}
	state.YAML = string(yamlData)
	state.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	state.History, err = s.History(ctx, profile)
	if err != nil {
		return ProfileState{}, err
	}
	if len(state.History) > 0 {
		state.PublishedHash = state.History[0].SHA256
	}
	state.Tokens, err = s.ListTokens(ctx, profile)
	return state, err
}

func (s *Store) Publish(ctx context.Context, profile, note string) (PublishedConfig, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PublishedConfig{}, err
	}
	defer tx.Rollback()

	var encryptedYAML []byte
	var hash string
	var current int
	err = tx.QueryRowContext(ctx, `SELECT draft_yaml, draft_hash, published_version
		FROM profiles WHERE name = ?`, profile).Scan(&encryptedYAML, &hash, &current)
	if err != nil {
		return PublishedConfig{}, err
	}
	yamlData, err := s.vault.decrypt(encryptedYAML)
	if err != nil {
		return PublishedConfig{}, err
	}
	version := current + 1
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO versions
		(profile, version, config_yaml, sha256, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		profile, version, encryptedYAML, hash, note, now.Format(time.RFC3339Nano)); err != nil {
		return PublishedConfig{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE profiles SET published_version = ? WHERE name = ?`, version, profile); err != nil {
		return PublishedConfig{}, err
	}
	if err := tx.Commit(); err != nil {
		return PublishedConfig{}, err
	}
	return PublishedConfig{Profile: profile, Version: version, YAML: yamlData, SHA256: hash, Note: note, Created: now}, nil
}

func (s *Store) Published(ctx context.Context, profile string) (PublishedConfig, error) {
	var result PublishedConfig
	var encrypted []byte
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT v.profile, v.version, v.config_yaml, v.sha256,
		v.note, v.created_at
		FROM versions v
		JOIN profiles p ON p.name = v.profile AND p.published_version = v.version
		WHERE v.profile = ?`, profile).
		Scan(&result.Profile, &result.Version, &encrypted, &result.SHA256, &result.Note, &created)
	if err != nil {
		return PublishedConfig{}, err
	}
	result.YAML, err = s.vault.decrypt(encrypted)
	if err != nil {
		return PublishedConfig{}, err
	}
	result.Created, _ = time.Parse(time.RFC3339Nano, created)
	return result, nil
}

func (s *Store) History(ctx context.Context, profile string) ([]Version, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT version, sha256, note, created_at
		FROM versions WHERE profile = ? ORDER BY version DESC LIMIT 50`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var versions []Version
	for rows.Next() {
		var version Version
		var created string
		if err := rows.Scan(&version.Version, &version.SHA256, &version.Note, &created); err != nil {
			return nil, err
		}
		version.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Store) RestoreVersionToDraft(ctx context.Context, profile string, version int) error {
	var encryptedYAML []byte
	err := s.db.QueryRowContext(ctx, `SELECT config_yaml FROM versions
		WHERE profile = ? AND version = ?`, profile, version).Scan(&encryptedYAML)
	if err != nil {
		return err
	}
	yamlData, err := s.vault.decrypt(encryptedYAML)
	if err != nil {
		return err
	}
	document, err := DocumentFromYAML(yamlData)
	if err != nil {
		return err
	}
	return s.SaveDraft(ctx, profile, document, yamlData)
}

func (s *Store) CreateToken(ctx context.Context, profile, name string) (TokenEntry, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return TokenEntry{}, "", err
	}
	token := "sshw_sync_" + base64.RawURLEncoding.EncodeToString(raw)
	hash := tokenHash(token)
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `INSERT INTO sync_tokens
		(name, profile, token_hash, created_at) VALUES (?, ?, ?, ?)`,
		name, profile, hash, now.Format(time.RFC3339Nano))
	if err != nil {
		return TokenEntry{}, "", err
	}
	id, _ := result.LastInsertId()
	return TokenEntry{ID: id, Name: name, Profile: profile, CreatedAt: now}, token, nil
}

func (s *Store) ValidateToken(ctx context.Context, profile, token string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sync_tokens
		WHERE profile = ? AND token_hash = ? AND revoked_at IS NULL`,
		profile, tokenHash(token)).Scan(&count)
	return count == 1, err
}

func (s *Store) ListTokens(ctx context.Context, profile string) ([]TokenEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, profile, created_at, revoked_at
		FROM sync_tokens WHERE profile = ? ORDER BY id DESC`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []TokenEntry
	for rows.Next() {
		var token TokenEntry
		var created string
		var revoked sql.NullString
		if err := rows.Scan(&token.ID, &token.Name, &token.Profile, &created, &revoked); err != nil {
			return nil, err
		}
		token.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		if revoked.Valid {
			value, _ := time.Parse(time.RFC3339Nano, revoked.String)
			token.RevokedAt = &value
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

func (s *Store) RevokeToken(ctx context.Context, profile string, id int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE sync_tokens SET revoked_at = ?
		WHERE id = ? AND profile = ? AND revoked_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339Nano), id, profile)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return errors.New("active token not found")
	}
	return nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
