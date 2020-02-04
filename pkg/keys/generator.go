package keys

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"

	"gopkg.in/square/go-jose.v2"
)

type jsonWebKeySetT struct {
	Keys []map[string]interface{} `json:"keys"`
}

func getPrivateKeyPem(privateKey *ecdsa.PrivateKey) (string, error) {
	ecPrivateKeyX509Encoded, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return "", err
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: ecPrivateKeyX509Encoded})), nil
}

func getKeyId(privateKey *ecdsa.PrivateKey) (string, error) {
	publicJSONWebKey := jose.JSONWebKey{
		Key: privateKey.Public(),
	}

	hash, err := publicJSONWebKey.Thumbprint(crypto.SHA256)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(hash), nil
}

func getPublicKeyPem(publicKey *ecdsa.PublicKey) (string, error) {
	publicKeyPKIXEncoded, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyPKIXEncoded})), nil
}

func getJwks(publicKey *ecdsa.PublicKey, keyId string) (string, error) {
	publicJSONWebKey := jose.JSONWebKey{
		Key:   publicKey,
		KeyID: keyId,
	}

	publicJSONWebKeyJsonBytes, err := json.Marshal(publicJSONWebKey)
	if err != nil {
		return "", err
	}

	var publicJSONWebKeyJsonMap map[string]interface{}
	err = json.Unmarshal(publicJSONWebKeyJsonBytes, &publicJSONWebKeyJsonMap)
	if err != nil {
		return "", err
	}

	publicKeyPem, err := getPublicKeyPem(publicKey)
	if err != nil {
		return "", err
	}

	publicJSONWebKeyJsonMap["pem"] = publicKeyPem

	jsonWebKeySet := jsonWebKeySetT{
		Keys: []map[string]interface{}{publicJSONWebKeyJsonMap},
	}

	jsonWebKeySetBytes, err := json.Marshal(jsonWebKeySet)
	if err != nil {
		return "", err
	}

	return string(jsonWebKeySetBytes), nil
}

func Generate() (privateKeyPem string, keyId string, jwks string, err error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return "", "", "", err
	}

	privateKeyPem, err = getPrivateKeyPem(privateKey)
	if err != nil {
		return "", "", "", err
	}

	keyId, err = getKeyId(privateKey)
	if err != nil {
		return "", "", "", err
	}

	jwks, err = getJwks(privateKey.Public().(*ecdsa.PublicKey), keyId)
	return privateKeyPem, keyId, jwks, nil
}
