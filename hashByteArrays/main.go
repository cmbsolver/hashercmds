package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
	"hash"
	"io/ioutil"
	"os"
	"strings"
)

func main() {
	// Check if the input is provided as a command-line argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <comma-separated byte values>")
		return
	}

	// Read input from the command-line argument
	input := os.Args[1]

	byteArray, err := convertToByteArray(input)
	if err != nil {
		fmt.Println("Error converting input to byte array:", err)
		return
	}

	existingHashString, err := readExistingHash("existinghash.txt")
	if err != nil {
		fmt.Println("Error reading existing hash:", err)
		return
	}

	hashFunctions := getHashFunctions()

	for name, hashFunc := range hashFunctions {
		hashString := computeHash(hashFunc, byteArray)
		if hashString == existingHashString {
			fmt.Printf("Match found with %s\n", name)
			appendToFile("found_hashes.txt", name, byteArray, hashString)
		}
	}
}

func convertToByteArray(input string) ([]byte, error) {
	byteValues := strings.Split(input, ",")
	byteArray := make([]byte, len(byteValues))
	for i, v := range byteValues {
		_, err := fmt.Sscanf(v, "%d", &byteArray[i])
		if err != nil {
			return nil, err
		}
	}
	return byteArray, nil
}

func readExistingHash(filename string) (string, error) {
	existingHash, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(existingHash)), nil
}

func getHashFunctions() map[string]func() hash.Hash {
	return map[string]func() hash.Hash{
		"SHA-512":  sha512.New,
		"SHA3-512": sha3.New512,
		"BLAKE2b":  func() hash.Hash { h, _ := blake2b.New512(nil); return h },
		// Add other hash functions here
	}
}

func computeHash(hashFunc func() hash.Hash, byteArray []byte) string {
	h := hashFunc()
	h.Write(byteArray)
	return hex.EncodeToString(h.Sum(nil))
}

func appendToFile(filename, hashName string, byteArray []byte, hashString string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	byteArrayString := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(byteArray)), ","), "[]")
	_, err = file.WriteString(fmt.Sprintf("Hash Name: %s\nByte Array: %s\nHash: %s\n\n", hashName, byteArrayString, hashString))
	if err != nil {
		fmt.Println("Error writing to file:", err)
	}
}
