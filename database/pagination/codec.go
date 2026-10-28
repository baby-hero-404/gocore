package pagination

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
)

const (
	cursorVersion = 1
	minKeyLen     = 32
)

// Codec signs and verifies cursor tokens with HMAC-SHA256.
type Codec struct {
	key []byte
}

// NewCodec returns a Codec using key, which must be at least 32 bytes.
func NewCodec(key []byte) (*Codec, error) {
	if len(key) < minKeyLen {
		return nil, stderr.Newf(stderr.CodeValidationFailed, "pagination: cursor key must be at least %d bytes", minKeyLen)
	}
	return &Codec{key: append([]byte(nil), key...)}, nil
}

type payload struct {
	V int     `json:"v"`
	S string  `json:"s"`
	K []typed `json:"k"`
}

// typed carries a value with its type tag so integers and times round-trip without loss.
type typed struct {
	T string `json:"t"`
	V string `json:"v"`
}

func errInvalidCursor() error {
	return stderr.New(stderr.CodeInvalidParam, "invalid cursor").WithKey("pagination.invalid_cursor")
}

func (c *Codec) sign(data []byte) []byte {
	m := hmac.New(sha256.New, c.key)
	m.Write(data)
	return m.Sum(nil)
}

func (c *Codec) encode(sorts []Sort, vals []any) (string, error) {
	p := payload{V: cursorVersion, S: fingerprint(sorts), K: make([]typed, len(vals))}
	for i, v := range vals {
		tv, err := encodeValue(v)
		if err != nil {
			return "", err
		}
		p.K[i] = tv
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return "", stderr.Wrap(stderr.CodeInternal, err, "pagination: encode cursor")
	}
	enc := base64.RawURLEncoding
	return enc.EncodeToString(raw) + "." + enc.EncodeToString(c.sign(raw)), nil
}

func (c *Codec) decode(token string, sorts []Sort) ([]any, error) {
	enc := base64.RawURLEncoding
	body, sig, ok := strings.Cut(token, ".")
	if !ok {
		return nil, errInvalidCursor()
	}
	raw, err := enc.DecodeString(body)
	if err != nil {
		return nil, errInvalidCursor()
	}
	mac, err := enc.DecodeString(sig)
	if err != nil || !hmac.Equal(mac, c.sign(raw)) {
		return nil, errInvalidCursor()
	}
	var p payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, errInvalidCursor()
	}
	if p.V != cursorVersion || p.S != fingerprint(sorts) || len(p.K) != len(sorts) {
		return nil, errInvalidCursor()
	}
	vals := make([]any, len(p.K))
	for i, tv := range p.K {
		v, err := decodeValue(tv)
		if err != nil {
			return nil, errInvalidCursor()
		}
		vals[i] = v
	}
	return vals, nil
}

func encodeValue(v any) (typed, error) {
	if t, ok := v.(time.Time); ok {
		return typed{T: "t", V: t.UTC().Format(time.RFC3339Nano)}, nil
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return typed{T: "i", V: strconv.FormatInt(rv.Int(), 10)}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return typed{T: "u", V: strconv.FormatUint(rv.Uint(), 10)}, nil
	case reflect.Float32, reflect.Float64:
		f := rv.Float()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return typed{}, stderr.New(stderr.CodeInternal, "pagination: NaN/Inf sort value is not supported")
		}
		return typed{T: "f", V: strconv.FormatFloat(f, 'g', -1, 64)}, nil
	case reflect.String:
		return typed{T: "s", V: rv.String()}, nil
	case reflect.Bool:
		return typed{T: "b", V: strconv.FormatBool(rv.Bool())}, nil
	default:
		return typed{}, stderr.New(stderr.CodeInternal, fmt.Sprintf("pagination: unsupported sort value type %T (NULL sort columns are not supported)", v))
	}
}

func decodeValue(tv typed) (any, error) {
	switch tv.T {
	case "i":
		return strconv.ParseInt(tv.V, 10, 64)
	case "u":
		return strconv.ParseUint(tv.V, 10, 64)
	case "f":
		return strconv.ParseFloat(tv.V, 64)
	case "s":
		return tv.V, nil
	case "b":
		return strconv.ParseBool(tv.V)
	case "t":
		return time.Parse(time.RFC3339Nano, tv.V)
	default:
		return nil, fmt.Errorf("unknown type tag %q", tv.T)
	}
}
