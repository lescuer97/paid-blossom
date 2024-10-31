package nostr

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	n "github.com/nbd-wtf/go-nostr"
	"strconv"
	"strings"
	"time"
)

const HeaderScheme = "Nostr"

const (
	AuthKind      = 24242
	BlossomAction = "t"
	Expiration    = "expiration"
)

// blossom action tags
type BlossomActionTag string

const (
	GET    = "get"
	UPLOAD = "upload"
	LIST   = "list"
	DELETE = "delete"
)

// Errors parsing event
var (
	ErrIncorrectKind        = errors.New("Incorrect error kind")
	ErrNoBlossomAction      = errors.New("No valid blossom action")
	ErrCreatedAtInTheFuture = errors.New("CreatedAt tag is in the future")
	ErrNoExpiration         = errors.New("No expiration tag")
	ErrEventExpired         = errors.New("Event expired")
	ErrInvalidSignature     = errors.New("Invalid Signature")
)

func ExpirationTagIsValid(tags n.Tags, now int64) (bool, error) {
	tag := ""
	for _, t := range tags {
		if t.Key() == Expiration {
			tag = t.Value()
			break
		}
	}
	if tag == "" {
		return false, ErrNoExpiration
	}
	exp, err := strconv.ParseInt(tag, 10, 64)
	if err != nil {
		return false, fmt.Errorf("strconv.ParseInt(tag, 10, 64 ). %w", err)
	}

	if exp < now {
		return false, nil
	}

	return true, nil
}

func ParseNostrHeader(authHeader string) (n.Event, error) {
	// Remove Nostr scheme
	separation := strings.Split(authHeader, " ")

	var nostrEvent n.Event

	jsonBytes, err := base64.URLEncoding.DecodeString(separation[1])
	if err != nil {
		return nostrEvent, fmt.Errorf(" Header %v. \n base64.URLEncoding.DecodeString(separation[1]). %w", separation[1], err)
	}

	err = json.Unmarshal(jsonBytes, &nostrEvent)
	if err != nil {
		return nostrEvent, fmt.Errorf("json.Unmarshal(jsonBytes, &nostrEvent). %w", err)
	}
	return nostrEvent, nil

}

func ValidateAuthEvent(event n.Event) error {
	valid, err := event.CheckSignature()
	if err != nil {
		return fmt.Errorf("event.CheckSignature(). %w", err)
	}

	if !valid {
		return ErrInvalidSignature
	}

	tags := event.Tags.FilterOut([]string{"x", Expiration, BlossomAction})
	now := time.Now().Unix()

	validExpiration, err := ExpirationTagIsValid(tags, now)
	if err != nil {
		return fmt.Errorf("ExpirationTagIsValid(tags, ). %w", err)
	}

	switch {
	case event.Kind != AuthKind:
		return ErrIncorrectKind
	case !event.Tags.ContainsAny(BlossomAction, []string{GET, UPLOAD, LIST, DELETE}):
		return ErrNoBlossomAction
	case event.CreatedAt.Time().Unix() > now:
		return ErrCreatedAtInTheFuture
	case !validExpiration:
		return ErrEventExpired
	}

	return nil
}

type NotifMessage struct {
	Message string `json:"message"`
}
