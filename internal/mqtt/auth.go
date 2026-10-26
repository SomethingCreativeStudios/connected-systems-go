package mqtt

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

type account struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"`
}

var bcryptHashFormat = regexp.MustCompile(`^\$2[aby]\$[0-9]{2}\$[./A-Za-z0-9]{53}$`)

func loadAccounts(path string, anonymous bool) (map[string][]byte, error) {
	users := make(map[string][]byte)
	if path == "" {
		if !anonymous {
			return nil, fmt.Errorf("MQTT users file is required")
		}
		return users, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open MQTT users: %w", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > 1<<20 {
		return nil, fmt.Errorf("MQTT users file exceeds 1 MiB")
	}
	decoder := json.NewDecoder(io.LimitReader(f, (1<<20)+1))
	decoder.DisallowUnknownFields()
	var entries []account
	if err = decoder.Decode(&entries); err != nil {
		return nil, fmt.Errorf("decode MQTT users: %w", err)
	}
	if entries == nil {
		return nil, fmt.Errorf("MQTT users file must contain an array")
	}
	var trailing any
	if err = decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("MQTT users file must contain one JSON array")
	}
	for _, entry := range entries {
		if entry.Username == "" || !utf8.ValidString(entry.Username) || strings.ContainsRune(entry.Username, 0) {
			return nil, fmt.Errorf("invalid MQTT username")
		}
		if _, exists := users[entry.Username]; exists {
			return nil, fmt.Errorf("duplicate MQTT username %q", entry.Username)
		}
		if _, err = bcrypt.Cost([]byte(entry.PasswordHash)); err != nil || !bcryptHashFormat.MatchString(entry.PasswordHash) {
			return nil, fmt.Errorf("invalid bcrypt hash for MQTT user %q", entry.Username)
		}
		users[entry.Username] = []byte(entry.PasswordHash)
	}
	if !anonymous && len(users) == 0 {
		return nil, fmt.Errorf("MQTT users file has no accounts")
	}
	return users, nil
}

// HashPassword never loads configuration, opens a database or starts listeners.
func HashPassword(input io.Reader, output io.Writer) error {
	b, err := io.ReadAll(io.LimitReader(input, 75))
	if err != nil {
		return err
	}
	b = []byte(strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r"))
	if len(b) == 0 || len(b) > 72 {
		return fmt.Errorf("password must contain 1 to 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword(b, 12)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(output, string(hash))
	return err
}
