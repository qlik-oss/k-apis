package keys

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	privateKeyPem, keyId, jwks, err := Generate()
	fmt.Printf("privateKeyPem: %v, keyId: %v, jwks: %v, err: %v\n", privateKeyPem, keyId, jwks, err)

	assert.NoError(t, err)

	assert.Contains(t, privateKeyPem, "-----BEGIN EC PRIVATE KEY-----")
	assert.Contains(t, privateKeyPem, "-----END EC PRIVATE KEY-----")

	assert.True(t, keyId != "")

	var jwksMap map[string]interface{}
	err = json.Unmarshal([]byte(jwks), &jwksMap)
	assert.NoError(t, err)

	jwksKeyMap := jwksMap["keys"].([]interface{})[0].(map[string]interface{})

	assert.Equal(t, keyId, jwksKeyMap["kid"])
	assert.Contains(t, jwksKeyMap["pem"], "-----BEGIN PUBLIC KEY-----")
	assert.Contains(t, jwksKeyMap["pem"], "-----END PUBLIC KEY-----")
}

func Test_getSelfSignedCertAndKey(t *testing.T) {
	commonName := "elastic.example"
	organization := "elastic-local-cert"
	validity := time.Hour * 24 * 365 * 10
	selfSignedCert, key, err := GetSelfSignedCertAndKey(commonName, organization, validity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fmt.Println(string(selfSignedCert))
	fmt.Println(string(key))

	block, _ := pem.Decode(selfSignedCert)
	if block == nil {
		t.Fatalf("unexpected error: %v", err)
	} else if cert, err := x509.ParseCertificate(block.Bytes); err != nil {
		t.Fatalf("unexpected error: %v", err)
	} else if cert.Subject.CommonName != commonName {
		t.Fatalf("unexpected error: %v", err)
	} else if !reflect.DeepEqual(cert.Subject.Organization, []string{organization}) {
		t.Fatalf("unexpected error: %v", err)
	} else if cert.Issuer.CommonName != commonName {
		t.Fatalf("unexpected error: %v", err)
	} else if !reflect.DeepEqual(cert.Issuer.Organization, []string{organization}) {
		t.Fatalf("unexpected error: %v", err)
	} else if !reflect.DeepEqual(cert.DNSNames, []string{commonName, fmt.Sprintf("*.%v", commonName)}) {
		t.Fatalf("unexpected error: %v", err)
	}
}
