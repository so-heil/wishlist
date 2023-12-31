package keystore

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var ErrInvalidKey = errors.New("requested key is not a valid keystore key")

type Key struct {
	Expire time.Time
	Signer crypto.Signer
}

type KeyStore struct {
	store            sync.Map
	rotationPeriod   time.Duration
	expirationPeriod time.Duration
	shutdown         <-chan os.Signal
	critErrs         chan<- error
	logger           *zap.SugaredLogger
	active           string
	SigningMethod    jwt.SigningMethod
}

// New creates a new keystore with one initial key, and starts key rotation
// When having multiple instances of the consumer application, keystore can exist on another service
func New(
	rotationPeriod time.Duration,
	expirationPeriod time.Duration,
	shutdown chan os.Signal,
	logger *zap.SugaredLogger,
) (*KeyStore, error) {
	ks := &KeyStore{
		rotationPeriod:   rotationPeriod,
		expirationPeriod: expirationPeriod,
		shutdown:         shutdown,
		logger:           logger,
		SigningMethod:    jwt.SigningMethodEdDSA,
	}

	// Init store with an initial key
	if err := ks.addKey(); err != nil {
		return nil, fmt.Errorf("add init key to store: %w", err)
	}

	ks.startRotation()

	return ks, nil
}

func (ks *KeyStore) Signer(id string) (Key, error) {
	val, ok := ks.store.Load(id)
	if !ok {
		return Key{}, ErrInvalidKey
	}
	key, ok := val.(Key)
	if !ok {
		err := fmt.Errorf("cannot assign map value to keystore key, map value: %v", val)
		ks.critErrs <- err
		return Key{}, err
	}
	return key, nil
}

func (ks *KeyStore) Active() (string, Key, error) {
	active := ks.active
	sig, err := ks.Signer(active)
	return active, sig, err
}

// Revoke is accessible to revoke keys when compromised
func (ks *KeyStore) Revoke(id string) {
	ks.store.Delete(id)
}

func (ks *KeyStore) startRotation() {
	ticker := time.NewTicker(ks.rotationPeriod)
	go func() {
		defer ticker.Stop()
		ks.logger.Infow("keystore rotation: starting", "rotation period", ks.rotationPeriod, "first rotation", time.Now().Add(ks.rotationPeriod))
		var round int
		for {
			select {
			case <-ks.shutdown:
				ks.logger.Infow("keystore rotation: shutting down", "rotations done", round)
				return
			case <-ticker.C:
				round++
				ks.logger.Infow("keystore rotation: starting", "round", round)
				expiredCount, err := ks.rotate()
				if err != nil {
					ks.critErrs <- fmt.Errorf("#%d keystore rotation: failed: %s", round, err)
				}
				ks.logger.Infow(
					"keystore rotation: successful",
					"expired in round",
					expiredCount,
					"round",
					round,
					"next rotation",
					time.Now().Add(ks.rotationPeriod),
				)
			}
		}
	}()
}

func (ks *KeyStore) rotate() (int, error) {
	var expiredCount int
	if err := ks.addKey(); err != nil {
		return expiredCount, fmt.Errorf("add new key on rotation: %w", err)
	}

	// Revoke expired keys
	var rangeErr error
	ks.store.Range(func(k any, v any) bool {
		id, ok := k.(string)
		if !ok {
			rangeErr = fmt.Errorf("range: cannot assign map key to string, map key: %v", k)
			return false
		}
		key, ok := v.(Key)
		if !ok {
			rangeErr = fmt.Errorf("range: cannot assign map value to keystore key, map value: %v", v)
			return false
		}
		if key.Expire.Before(time.Now()) {
			ks.Revoke(id)
			expiredCount++
		}
		return true
	})

	if rangeErr != nil {
		return expiredCount, fmt.Errorf("range on keys: %w", rangeErr)
	}

	return expiredCount, nil
}

func (ks *KeyStore) addKey() error {
	id, newKey, err := genKey()
	if err != nil {
		return fmt.Errorf("generate new key: %w", err)
	}
	ks.store.Store(id, Key{
		Expire: time.Now().Add(ks.expirationPeriod),
		Signer: newKey,
	})
	ks.active = id
	return nil
}

func genKey() (string, crypto.Signer, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil, fmt.Errorf("ed25519 gen key: %w", err)
	}

	uid, err := uuid.NewUUID()
	if err != nil {
		return "", nil, fmt.Errorf("new uuid: %w", err)
	}

	return uid.String(), priv, nil
}
