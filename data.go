package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"log"
	"math/big"

	badger "github.com/dgraph-io/badger/v3"
	"golang.org/x/crypto/bcrypt"
)

type Record struct {
	Shortlink    string
	RedirectTo   string
	PasswordHash []byte
}

var ErrorShortlinkNotFound = errors.New("shortlink not found")
var ErrorShortlinkAlreadyExists = errors.New("shortlink already exists")
var ErrorPasswordIncorrect = errors.New("password incorrect")

// from: https://itnext.io/encrypt-data-with-a-password-in-go-b5366384e291
func encrypt(key, data []byte) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	return ciphertext, nil
}

func decrypt(key, data []byte) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func calculateEncryptionKey(shortlink *string) []byte {
	hash := sha256.New()
	hash.Write([]byte(*shortlink))
	hash.Write([]byte(*shortlink))
	shortlink_hash := hash.Sum(nil)
	return shortlink_hash[:32]
}

func calculateDbKey(shortlink *string) []byte {
	hash := sha256.New()
	hash.Write([]byte(*shortlink))
	shortlink_hash := hash.Sum(nil)
	hex_digest := hex.EncodeToString(shortlink_hash)
	return append([]byte("shortlink_"), []byte(hex_digest)...)
}

func encodeRecord(rec *Record) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(rec)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return buf.Bytes(), nil

}

func AddRecord(db *badger.DB, shortlink, redirect, password *string) bool {
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
		return false
	}
	rec := Record{*shortlink, *redirect, hash}
	serialized, err := encodeRecord(&rec)
	if err != nil {
		log.Fatal(err)
		return false
	}
	encrypted, err := encrypt(calculateEncryptionKey(shortlink), serialized)
	if err != nil {
		log.Fatal(err)
		return false
	}
	insertAt := calculateDbKey(shortlink)
	err = db.Update(func(txn *badger.Txn) error {
		_, err := txn.Get(insertAt)
		if err == badger.ErrKeyNotFound {
			return txn.Set(insertAt, encrypted)
		} else {
			return ErrorShortlinkAlreadyExists
		}
	})
	if err != nil {
		log.Fatal(err)
		return false
	}
	return true
}

func DoesShortlinkExist(db *badger.DB, shortlink *string) bool {
	key := calculateDbKey(shortlink)
	err := db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	if err == badger.ErrKeyNotFound {
		return false
	}
	if err != nil {
		log.Fatal(err)
		return false
	}
	return true
}

var alphabet = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789-_")

func IsValidShortlink(shortlink *string) bool {
	if len(*shortlink) < 3 || len(*shortlink) > 32 {
		return false
	}
	for _, c := range *shortlink {
		// yes this is O(n^2), but it won't really matter
		found := false
		for _, a := range alphabet {
			if a == c {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func GenerateRandomShortlink(db *badger.DB) string {

	for {
		b := make([]rune, 8)
		for i := range b {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
			if err != nil {
				log.Fatal(err)
			}
			b[i] = alphabet[n.Int64()]
		}
		shortlink := string(b)
		if !DoesShortlinkExist(db, &shortlink) {
			return shortlink
		}
	}
}

func GetRecord(db *badger.DB, shortlink *string) (*Record, error) {
	var encoded []byte
	key := calculateDbKey(shortlink)
	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		var err2 error
		encoded, err2 = item.ValueCopy(nil)
		if err2 != nil {
			return err
		}
		return nil
	})
	if err == badger.ErrKeyNotFound {
		return nil, ErrorShortlinkNotFound
	}
	if err != nil {
		return nil, err
	}
	decrypted, err := decrypt(calculateEncryptionKey(shortlink), encoded)
	if err != nil {
		return nil, err
	}
	var rec Record
	buf := bytes.NewBuffer(decrypted)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&rec)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func DeleteRecord(db *badger.DB, shortlink, password *string) error {
	rec, err := GetRecord(db, shortlink)
	if err != nil {
		return ErrorShortlinkNotFound
	}
	err = bcrypt.CompareHashAndPassword(rec.PasswordHash, []byte(*password))
	if err != nil {

		return ErrorPasswordIncorrect
	}
	key := calculateDbKey(shortlink)
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func DeleteRecordDisregardingPassword(db *badger.DB, shortlink *string) error {
	key := calculateDbKey(shortlink)
	err := db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}
