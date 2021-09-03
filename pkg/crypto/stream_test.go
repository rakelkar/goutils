/*
Copyright (c) 2021 - Present. Blend Labs, Inc. All rights reserved
Use of this source code is governed by a MIT license that can be found in the LICENSE file.
*/

package crypto

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
    letterIdxBits = 6                    // 6 bits to represent a letter index
    letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
    letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)
var src = rand.NewSource(time.Now().UnixNano())
func randStringBytesMaskImprSrcSB(n int) string {
    sb := strings.Builder{}
    sb.Grow(n)
    // A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
    for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
        if remain == 0 {
            cache, remain = src.Int63(), letterIdxMax
        }
        if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
            sb.WriteByte(letterBytes[idx])
            i--
        }
        cache >>= letterIdxBits
        remain--
    }

    return sb.String()
}

func Test_Stream_EncrypterDecrypter(t *testing.T) {
	t.Parallel()

	encKey, err := CreateKey(32)
	macKey, err := CreateKey(32)
	plaintext := "Eleven is the best person in all of Hawkins Indiana. Some more text"
	pt := []byte(plaintext)

	src := bytes.NewReader(pt)

	se, err := NewStreamEncrypter(encKey, macKey, src)
	assert.Nil(t, err)

	encrypted, err := ioutil.ReadAll(se)
	assert.Nil(t, err)

	sd, err := NewStreamDecrypter(encKey, macKey, se.Meta(), bytes.NewReader(encrypted))
	assert.Nil(t, err)

	decrypted, err := ioutil.ReadAll(sd)
	assert.Equal(t, plaintext, string(decrypted))

	assert.Nil(t, sd.Authenticate())
}

func BenchmarkEncrypterDecrypter(t *testing.B) {

	encKey, err := CreateKey(32)
	macKey, err := CreateKey(32)

	plaintext := randStringBytesMaskImprSrcSB(1048577*1000)
	//plaintext := "Eleven is the best person in all of Hawkins Indiana. Some more text"
	pt := []byte(plaintext)

	src := bytes.NewReader(pt)

	se, err := NewStreamEncrypter(encKey, macKey, src)
	assert.Nil(t, err)

	encrypted, err := ioutil.ReadAll(se)
	assert.Nil(t, err)

	sd, err := NewStreamDecrypter(encKey, macKey, se.Meta(), bytes.NewReader(encrypted))
	assert.Nil(t, err)

	decrypted, err := ioutil.ReadAll(sd)
	assert.Equal(t, plaintext, string(decrypted))

	assert.Nil(t, sd.Authenticate())
}

func BenchmarkBaseline(t *testing.B) {

	CreateKey(32)
	CreateKey(32)

	plaintext := randStringBytesMaskImprSrcSB(1048577)
	pt := []byte(plaintext)

	src := bytes.NewReader(pt)

	encrypted, err := ioutil.ReadAll(src)
	assert.Nil(t, err)

	decrypted, err := ioutil.ReadAll(bytes.NewReader(encrypted))
	assert.Equal(t, plaintext, string(decrypted))
}

