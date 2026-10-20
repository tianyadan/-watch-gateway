package bus

import (
	"crypto/aes"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	signSalt = "qwihrnbtmj"

	// Python 中：
	//
	// AES_KEY.encode("utf-8").hex()
	//
	// 实际使用的 Key 就是下面这 32 个 ASCII 字节。
	aesKey = "FF32AE65FBFD19414EAAFF6291A54B42"
)

// param 不能换成 map。
// 因为 cryptoSign 的计算依赖参数顺序。
type param struct {
	Key   string
	Value string
}

// cryptoSign 对应 Python:
//
// source = "&".join(
//
//	f'"{key}"="{value}"'
//	for key, value in params.items()
//
// )
//
// md5(source + SIGN_SALT)
func cryptoSign(params []param) string {

	var builder strings.Builder

	for i, item := range params {

		if i > 0 {
			builder.WriteByte('&')
		}

		builder.WriteByte('"')
		builder.WriteString(item.Key)

		builder.WriteString("\"=\"")

		builder.WriteString(item.Value)

		builder.WriteByte('"')
	}

	source := builder.String() + signSalt

	sum := md5.Sum([]byte(source))

	return hex.EncodeToString(sum[:])
}

// decryptPayload
//
// 对应 Python:
//
// openssl enc
// -d
// -aes-256-ecb
// -K xxx
// -a
// -A
func decryptPayload(ciphertext string) ([]byte, error) {

	// OpenSSL -a 表示 Base64
	decoded, err := base64.StdEncoding.DecodeString(
		strings.TrimSpace(ciphertext),
	)

	if err != nil {

		// 某些 Base64 数据可能没有 padding，
		// 再尝试 RawStdEncoding。
		decoded, err = base64.RawStdEncoding.DecodeString(
			strings.TrimSpace(ciphertext),
		)

		if err != nil {
			return nil, fmt.Errorf(
				"base64 decode failed: %w",
				err,
			)
		}
	}

	// AES-256 要求 32 字节 Key
	block, err := aes.NewCipher([]byte(aesKey))

	if err != nil {
		return nil, fmt.Errorf(
			"create aes cipher failed: %w",
			err,
		)
	}

	blockSize := block.BlockSize()

	if len(decoded) == 0 ||
		len(decoded)%blockSize != 0 {

		return nil, errors.New(
			"ciphertext length is not a multiple of AES block size",
		)
	}

	plaintext := make([]byte, len(decoded))

	// Go 标准库没有直接提供 ECB Mode。
	//
	// ECB 的本质就是：
	//
	// 每 16 bytes 独立 AES 解密一次。
	for start := 0; start < len(decoded); start += blockSize {

		block.Decrypt(
			plaintext[start:start+blockSize],
			decoded[start:start+blockSize],
		)
	}

	// OpenSSL 默认使用 PKCS#7 padding。
	plaintext, err = pkcs7Unpad(
		plaintext,
		blockSize,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"pkcs7 unpad failed: %w",
			err,
		)
	}

	return plaintext, nil
}

// 删除 PKCS#7 Padding。
func pkcs7Unpad(
	data []byte,
	blockSize int,
) ([]byte, error) {

	if len(data) == 0 ||
		len(data)%blockSize != 0 {

		return nil, errors.New(
			"invalid padded data length",
		)
	}

	padding := int(
		data[len(data)-1],
	)

	if padding <= 0 ||
		padding > blockSize ||
		padding > len(data) {

		return nil, errors.New(
			"invalid padding size",
		)
	}

	// 检查 padding 是否正确。
	for _, b := range data[len(data)-padding:] {

		if int(b) != padding {
			return nil, errors.New(
				"invalid padding bytes",
			)
		}
	}

	return data[:len(data)-padding], nil
}
