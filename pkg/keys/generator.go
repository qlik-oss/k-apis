package keys

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

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

func GetSelfSignedCertAndKey(name, organization string, validity time.Duration) (certificate, key []byte, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("ailed to generate serial number: %s", err)
	}
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   name,
			Organization: []string{organization},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{name, fmt.Sprintf("*.%v", name)},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %s", err)
	}
	certificate = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal private key: %v", err)
	}
	key = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	return certificate, key, nil
}
