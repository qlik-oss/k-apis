package keys

import (
	"encoding/json"
	"fmt"
	"testing"

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
