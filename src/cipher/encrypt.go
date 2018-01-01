package cipher

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	// the data size of each block
	blockSize = 32 // 32 bytes
	// nonce data size
	nonceSize = 32 // 32 bytes
	// checksum data size
	checksumSize = 32 // 32 bytes
	// the data length prefix size
	prefixLengthSize = 4 // 4 bytes
)

var (
	// ErrInvalidPassword represents the invalid password error
	ErrInvalidPassword = errors.New("invalid password")
)

// Encrypt encrypts the data with password
//
// 1> Add 32 bits length prefix to indicate the length of data. <length(4 bytes)><data>
// 2> Pad the length + data to 32 bytes will nulls at end
// 2> SHA256(<length(4 bytes)><data><padding>) and prefix the hash. <hash(32 bytes)><length(4 bytes)><data><padding>
// 3> Split the whole data(hash+length+data+padding) into 256 bits(32 bytes) blocks
// 4> Each block is encrypted by XORing the unencrypted block with SHA256(SHA256(password), SHA256(index, SHA256(nonce))
// 	  - index is 0 for the first block of 32 bytes, 1 for the second block of 32 bytes, 2 for third block
// 5> Prefix nonce and SHA256 the nonce with blocks to get checksum, and prefix the checksum
// 6> Finally, the data format is: <checksum(32 bytes)><nonce(32 bytes)><block0.Hex(), block1.Hex()...>
func Encrypt(data []byte, password []byte) ([]byte, error) {
	// set data length prefix
	length := make([]byte, prefixLengthSize)
	binary.PutUvarint(length, uint64(len(data))) // the length
	var pbuf bytes.Buffer
	pbuf.Write(length)
	pbuf.Write(data)
	pdata := pbuf.Bytes()

	// Pad length + data with null to 32 bytes
	l := len(pdata) // hash + length + data
	n := l / blockSize
	m := l % blockSize
	if m > 0 {
		paddingNull := make([]byte, blockSize-m)
		pdata = append(pdata, paddingNull...)
		n++
	}

	// Hash(length+data+padding)
	dataHash := SumSHA256(pdata)

	// Initialize blocks with data hash
	blocks := [][blockSize]byte{dataHash}
	for i := 0; i < n; i++ {
		var b [blockSize]byte
		copy(b[:], pdata[i*blockSize:(i+1)*blockSize])
		blocks = append(blocks, b)
	}

	// Generates a nonce
	nonce := RandByte(nonceSize)
	// Hash the nonce
	hashNonce := SumSHA256(nonce)
	// Hash the password
	hashPassword := SumSHA256(password)

	var encryptedData []byte
	// Encodes the blocks
	for i := range blocks {
		// Hash(password, hash(index, hash(nonce)))
		h := hashPwdIndexNonce(hashPassword, int64(i), hashNonce)
		// Converts the block to SHA256
		bh := SHA256(blocks[i])
		encryptedHash := bh.Xor(h)
		encryptedData = append(encryptedData, encryptedHash[:]...)
	}

	// Prefix the nonce
	nonceAndDataBytes := append(nonce, encryptedData...)
	// Calculates the checksum
	checkSum := SumSHA256(nonceAndDataBytes)
	var buf bytes.Buffer
	_, err := buf.Write(checkSum[:])
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(nonceAndDataBytes)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decrypt decrypts the data
func Decrypt(data []byte, password []byte) ([]byte, error) {
	buf := bytes.NewBuffer(data)

	// Gets checksum
	checkSumBytes := make([]byte, checksumSize)
	n, err := buf.Read(checkSumBytes)
	if err != nil {
		return nil, err
	}

	if n != checksumSize {
		return nil, errors.New("invalid checksum length")
	}

	var checkSum SHA256
	copy(checkSum[:], checkSumBytes)

	// Checks the checksum
	csh := SumSHA256(buf.Bytes())
	if csh != checkSum {
		return nil, errors.New("invalid checksum")
	}

	// Gets the nonce
	nonce := make([]byte, nonceSize)
	n, err = buf.Read(nonce)
	if err != nil {
		return nil, err
	}

	if n != nonceSize {
		return nil, errors.New("invalid nonce length")
	}

	var decodeData []byte
	hashPassword := SumSHA256(password)
	hashNonce := SumSHA256(nonce)
	i := 0
	for {
		var block SHA256
		n, err := buf.Read(block[:])
		if err == io.EOF {
			break
		}

		if n != blockSize {
			return nil, errors.New("invalid block size, must be multiple of 32 bytes")
		}

		// Decodes the block
		dataHash := block.Xor(hashPwdIndexNonce(hashPassword, int64(i), hashNonce))
		decodeData = append(decodeData, dataHash[:]...)
		i++
	}

	buf = bytes.NewBuffer(decodeData)

	// Gets the hash
	var dataHash SHA256
	n, err = buf.Read(dataHash[:])
	if err != nil {
		return nil, fmt.Errorf("read data hash failed: %v", err)
	}

	if n != 32 {
		return nil, errors.New("read data hash failed: read length != 32")
	}

	// Checks the hash
	if dataHash != SumSHA256(buf.Bytes()) {
		return nil, ErrInvalidPassword
	}

	// Reads out the prefix length
	lb := make([]byte, prefixLengthSize)
	n, err = buf.Read(lb)
	if err != nil {
		return nil, err
	}

	if n != prefixLengthSize {
		return nil, errors.New("read prefix length failed")
	}

	l, _ := binary.Uvarint(lb)
	if l > uint64(buf.Len()) {
		return nil, errors.New("invalid prefix length")
	}

	// Reads out the raw data
	rawData := make([]byte, l)
	n, err = buf.Read(rawData)
	if err != nil {
		return nil, err
	}

	if uint64(n) != l {
		return nil, fmt.Errorf("read data failed, expect %d bytes, but get %d bytes", l, n)
	}

	return rawData, nil
}

// hash(password, hash(index, hash(nonce)))
func hashPwdIndexNonce(password SHA256, index int64, nonceHash SHA256) SHA256 {
	// convert index to 256bit number
	indexBytes := make([]byte, 32)
	binary.PutVarint(indexBytes, index)

	// hash(index, nonceHash)
	indexNonceHash := SumSHA256(append(indexBytes, nonceHash[:]...))

	// hash(hash(password), indexNonceHash)
	return AddSHA256(password, indexNonceHash)
}
