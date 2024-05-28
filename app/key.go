package app

import (
	app_db "commune/db/gen"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"io"
)

func generateKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func encryptPrivateKey(privateKey *rsa.PrivateKey, passphrase []byte) (string, error) {
	block, err := aes.NewCipher(passphrase)
	if err != nil {
		return "", err
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	encrypted := gcm.Seal(nonce, nonce, privateKeyBytes, nil)
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func decryptPrivateKey(encryptedPrivateKey string, passphrase []byte) (*rsa.PrivateKey, error) {
	encryptedBytes, err := base64.StdEncoding.DecodeString(encryptedPrivateKey)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(passphrase)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedBytes) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := encryptedBytes[:nonceSize], encryptedBytes[nonceSize:]
	privateKeyBytes, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func publicKeyToPEM(pubKey *rsa.PublicKey) (string, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return "", err
	}
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})
	return string(pubKeyPEM), nil
}

func (c *App) CreateNewUserKey(mid string) (*string, error) {
	priv, pub, err := generateKeyPair(2048)
	if err != nil {
		return nil, err
	}

	pkey := x509.MarshalPKCS1PublicKey(pub)

	enckey, err := encryptPrivateKey(priv, []byte(c.Config.App.EncryptionPassphrase))
	if err != nil {
		return nil, err
	}

	err = c.DB.Queries.CreateUserKey(context.Background(), app_db.CreateUserKeyParams{
		MatrixUserID: mid,
		PublicKey:    pkey,
		PrivateKey:   enckey,
	})

	if err != nil {
		return nil, err
	}

	pem, err := publicKeyToPEM(pub)
	if err != nil {
		return nil, err
	}

	return &pem, nil
}
