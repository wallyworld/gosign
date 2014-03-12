//
// gosign - Go HTTP signing library for the Joyent Public Cloud and Joyent Manta
//
//
// Copyright (c) 2013 Joyent Inc.
//
// Written by Daniele Stroppa <daniele.stroppa@joyent.com>
//

package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	// Authorization Headers
	SdcSignature   = "Signature keyId=\"/%s/keys/%s\",algorithm=\"%s\" %s"
	MantaSignature = "Signature keyId=\"/%s/keys/%s\",algorithm=\"%s\",signature=\"%s\""
)

type Endpoint struct {
	URL string
}

type Auth struct {
	User      string
	KeyFile   string
	Algorithm string
}

type Credentials struct {
	UserAuthentication Auth
	SdcKeyId           string
	SdcEndpoint        Endpoint
	MantaKeyId         string
	MantaEndpoint      Endpoint
}

type PrivateKey struct {
	key *rsa.PrivateKey
}

// The CreateAuthorizationHeader returns the Authorization header for the give request.
func CreateAuthorizationHeader(headers http.Header, credentials *Credentials, isMantaRequest bool) (string, error) {
	if isMantaRequest {
		signature, err := GetSignature(&credentials.UserAuthentication, "date: "+headers.Get("Date"))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(MantaSignature, credentials.UserAuthentication.User, credentials.MantaKeyId,
			credentials.UserAuthentication.Algorithm, signature), nil
	}
	signature, err := GetSignature(&credentials.UserAuthentication, headers.Get("Date"))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(SdcSignature, credentials.UserAuthentication.User, credentials.SdcKeyId,
		credentials.UserAuthentication.Algorithm, signature), nil
}

// The GetSignature method signs the specified key according to http://apidocs.joyent.com/cloudapi/#issuing-requests
// and http://apidocs.joyent.com/manta/api.html#authentication.
func GetSignature(auth *Auth, signing string) (string, error) {
	key, err := ioutil.ReadFile(auth.KeyFile)
	if err != nil {
		return "", fmt.Errorf("An error occurred while reading the key: %s", err)
	}
	block, _ := pem.Decode(key)
	rsakey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("An error occurred while parsing the key: %s", err)
	}
	privateKey := &PrivateKey{rsakey}

	hashFunc := getHashFunction(auth.Algorithm)
	hash := hashFunc.New()
	hash.Write([]byte(signing))

	digest := hash.Sum(nil)

	signed, err := rsa.SignPKCS1v15(rand.Reader, privateKey.key, hashFunc, digest)
	if err != nil {
		return "", fmt.Errorf("An error occurred while signing the key: %s", err)
	}

	return base64.StdEncoding.EncodeToString(signed), nil
}

// Helper method to get the Hash function based on the algorithm
func getHashFunction(algorithm string) (hashFunc crypto.Hash) {
	switch strings.ToLower(algorithm) {
	case "rsa-sha1":
		hashFunc = crypto.SHA1
	case "rsa-sha224", "rsa-sha256":
		hashFunc = crypto.SHA256
	case "rsa-sha384", "rsa-sha512":
		hashFunc = crypto.SHA512
	default:
		hashFunc = crypto.SHA256
	}
	return
}
