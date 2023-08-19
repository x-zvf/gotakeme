package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"flag"
	"fmt"
	"log"

	badger "github.com/dgraph-io/badger/v3"
	"golang.org/x/crypto/bcrypt"
)

type Record struct {
	Shortlink    string
	RedirectTo   string
	PasswordHash []byte
}

// from: https://itnext.io/encrypt-data-with-a-password-in-go-b5366384e291
func Encrypt(key, data []byte) ([]byte, error) {
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

func Decrypt(key, data []byte) ([]byte, error) {
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

func encryptionKey(shortlink string) []byte {
	hash := sha256.New()
	hash.Write([]byte(shortlink))
	hash.Write([]byte(shortlink))
	shortlink_hash := hash.Sum(nil)
	return shortlink_hash[:32]
}

func dbKey(shortlink string) []byte {
	hash := sha256.New()
	hash.Write([]byte(shortlink))
	shortlink_hash := hash.Sum(nil)
	return shortlink_hash
}

func encodeRecord(rec Record) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(rec)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func addRecord(db *badger.DB, shortlink, redirect, password string) bool {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
		return false
	}
	rec := Record{shortlink, redirect, hash}
	serialized, err := encodeRecord(rec)
	if err != nil {
		log.Fatal(err)
		return false
	}
	encrypted, err := Encrypt(encryptionKey(shortlink), serialized)
	if err != nil {
		log.Fatal(err)
		return false
	}
	insertAt := dbKey(shortlink)
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(insertAt, encrypted)
	})
	if err != nil {
		log.Fatal(err)
		return false
	}
	return true
}

func getRecord(db *badger.DB, shortlink string) (*Record, error) {
	var encoded []byte
	key := dbKey(shortlink)
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
	fmt.Println("got key")
	if err != nil {
		return nil, err
	}
	decrypted, err := Decrypt(encryptionKey(shortlink), encoded)
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

func main() {
	db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
	if err != nil {
		log.Fatal(err)
		return
	}
	defer db.Close()

	shortlink := flag.String("s", "shortlink", "Shortlink to add")
	target := flag.String("t", "target", "Target URL")
	delete_password := flag.String("p", "password", "Password to delete the shortlink")
	get := flag.Bool("get", false, "Get the shortlink")
	flag.Parse()
	if *get {
		rec, err := getRecord(db, *shortlink)
		if err != nil {
			log.Fatal(err)
			return
		}
		log.Printf("Shortlink: %s\nRedirect: %s\n", rec.Shortlink, rec.RedirectTo)
		return
	}

	if *shortlink == "" || *target == "" || *delete_password == "" {
		log.Fatal("Missing required arguments")
		return
	}
	addRecord(db, *shortlink, *target, *delete_password)

}
