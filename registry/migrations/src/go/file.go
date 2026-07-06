package migrations

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

const (
	maxMigrationNameLen = 128
	versionDigits       = 14
)

// Migration is a paired up/down SQL migration loaded from fs.FS.
type Migration struct {
	Version  int64
	Name     string
	UpSQL    string
	DownSQL  string
	Checksum string
}

// Load reads and validates migrations from fsys using Defaults().
func Load(fsys fs.FS, opts ...Options) ([]Migration, error) {
	opt := Defaults()
	if len(opts) > 0 {
		opt = opts[0]
	}
	opt, err := opt.normalize()
	if err != nil {
		return nil, err
	}
	return loadMigrations(fsys, opt)
}

func loadMigrations(fsys fs.FS, opt Options) ([]Migration, error) {
	if fsys == nil {
		return nil, errors.New("migrations: fsys is nil")
	}
	entries, err := fs.ReadDir(fsys, opt.Dir)
	if err != nil {
		return nil, fmt.Errorf("migrations: read dir: %w", err)
	}
	if len(entries) > opt.MaxMigrations*2 {
		return nil, errors.New("migrations: too many files in migration directory")
	}

	byVersion := make(map[int64]*Migration)
	seenFiles := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		base := entry.Name()
		direction := ""
		switch {
		case strings.HasSuffix(base, ".up.sql"):
			direction = "up"
		case strings.HasSuffix(base, ".down.sql"):
			direction = "down"
		default:
			continue
		}
		seenFiles++
		if seenFiles > opt.MaxMigrations*2 {
			return nil, errors.New("migrations: too many migration files")
		}
		version, name, err := parseMigrationName(base, direction)
		if err != nil {
			return nil, err
		}
		sqlText, err := readMigrationSQL(fsys, opt, base)
		if err != nil {
			return nil, err
		}
		m := byVersion[version]
		if m == nil {
			m = &Migration{Version: version, Name: name}
			byVersion[version] = m
		}
		if m.Name != name {
			return nil, fmt.Errorf("migrations: version %d has conflicting names", version)
		}
		if direction == "up" {
			if m.UpSQL != "" {
				return nil, fmt.Errorf("migrations: duplicate up migration for version %d", version)
			}
			m.UpSQL = sqlText
			m.Checksum = checksumSQL(sqlText)
		} else {
			if m.DownSQL != "" {
				return nil, fmt.Errorf("migrations: duplicate down migration for version %d", version)
			}
			m.DownSQL = sqlText
		}
	}

	migrations := make([]Migration, 0, len(byVersion))
	for _, m := range byVersion {
		if m.UpSQL == "" {
			return nil, fmt.Errorf("migrations: version %d is missing an up migration", m.Version)
		}
		migrations = append(migrations, *m)
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

func parseMigrationName(base, direction string) (int64, string, error) {
	if containsUnsafe(base) {
		return 0, "", errors.New("migrations: file name contains unsafe characters")
	}
	suffix := "." + direction + ".sql"
	stem := strings.TrimSuffix(base, suffix)
	if len(stem) <= versionDigits+1 || stem[versionDigits] != '_' {
		return 0, "", fmt.Errorf("migrations: invalid file name %q", base)
	}
	rawVersion := stem[:versionDigits]
	for i := 0; i < len(rawVersion); i++ {
		if rawVersion[i] < '0' || rawVersion[i] > '9' {
			return 0, "", fmt.Errorf("migrations: invalid version in %q", base)
		}
	}
	version, err := strconv.ParseInt(rawVersion, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("migrations: invalid version in %q: %w", base, err)
	}
	name := stem[versionDigits+1:]
	if err := validateMigrationName(name); err != nil {
		return 0, "", fmt.Errorf("migrations: invalid file name %q: %w", base, err)
	}
	return version, name, nil
}

func readMigrationSQL(fsys fs.FS, opt Options, base string) (string, error) {
	full := path.Join(opt.Dir, base)
	data, err := fs.ReadFile(fsys, full)
	if err != nil {
		return "", fmt.Errorf("migrations: read %s: %w", base, err)
	}
	if int64(len(data)) > opt.MaxSQLBytes {
		return "", fmt.Errorf("migrations: %s exceeds max SQL size", base)
	}
	if strings.ContainsRune(string(data), '\x00') {
		return "", fmt.Errorf("migrations: %s contains a null byte", base)
	}
	sqlText := strings.TrimSpace(string(data))
	if sqlText == "" {
		return "", fmt.Errorf("migrations: %s is empty", base)
	}
	return sqlText, nil
}

func checksumSQL(sqlText string) string {
	sum := sha256.Sum256([]byte(sqlText))
	return hex.EncodeToString(sum[:])
}

func validateDir(dir string) error {
	if containsUnsafe(dir) {
		return errors.New("migrations: directory contains unsafe characters")
	}
	if strings.Contains(dir, "..") {
		return errors.New("migrations: directory contains path traversal")
	}
	clean := path.Clean(dir)
	if clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
		return errors.New("migrations: directory must stay within fs root")
	}
	return nil
}

func validateMigrationName(name string) error {
	if name == "" {
		return errors.New("name is empty")
	}
	if len(name) > maxMigrationNameLen {
		return errors.New("name is too long")
	}
	if strings.Contains(name, "..") {
		return errors.New("name contains path traversal")
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			continue
		}
		return errors.New("name contains unsafe characters")
	}
	return nil
}

func validateTableName(table string) error {
	if table == "" {
		return errors.New("migrations: table name is empty")
	}
	if len(table) > maxMigrationNameLen {
		return errors.New("migrations: table name is too long")
	}
	if containsUnsafe(table) || strings.Contains(table, "..") {
		return errors.New("migrations: table name contains unsafe characters")
	}
	parts := strings.Split(table, ".")
	if len(parts) > 3 {
		return errors.New("migrations: table name has too many segments")
	}
	for _, part := range parts {
		if !isIdentifier(part) {
			return errors.New("migrations: table name contains unsafe identifier")
		}
	}
	return nil
}

func containsUnsafe(s string) bool {
	return strings.ContainsAny(s, "\r\n\x00")
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '_':
			continue
		case i == 0 && isASCIILetter(c):
			continue
		case i > 0 && (isASCIILetter(c) || isASCIIDigit(c)):
			continue
		default:
			return false
		}
	}
	return true
}

func isASCIILetter(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}

func isASCIIDigit(c byte) bool {
	return '0' <= c && c <= '9'
}
