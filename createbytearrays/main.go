package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
)

type Program struct {
	tasks chan []byte
}

func NewProgram() *Program {
	return &Program{
		tasks: make(chan []byte, 10000), // Increase buffer size
	}
}

func (p *Program) GenerateAllByteArrays(maxArrayLength int) {
	p.GenerateByteArrays(maxArrayLength, 1, nil)
	close(p.tasks)
}

func (p *Program) GenerateByteArrays(maxArrayLength, currentArrayLevel int, passedArray []byte) {
	if currentArrayLevel == maxArrayLength {
		currentArray := make([]byte, currentArrayLevel)
		if passedArray != nil {
			copy(currentArray, passedArray)
		}
		for i := 0; i < 256; i++ {
			currentArray[currentArrayLevel-1] = byte(i)
			p.tasks <- append([]byte(nil), currentArray...) // Send a copy to avoid data race
		}
	} else {
		currentArray := make([]byte, currentArrayLevel)
		if passedArray != nil {
			copy(currentArray, passedArray)
		}
		for i := 0; i < 256; i++ {
			currentArray[currentArrayLevel-1] = byte(i)
			p.GenerateByteArrays(maxArrayLength, currentArrayLevel+1, currentArray)
		}
	}
}

func processTasks(tasks chan []byte, wg *sync.WaitGroup, existingHash string) {
	defer wg.Done()

	// Open the file in append mode
	file, err := os.OpenFile("found_hashes.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	buffer := make([]byte, 0, 4096) // Buffer for batching writes

	for task := range tasks {
		hashes := generateHashes(task)
		for hashName, hash := range hashes {
			if hash == existingHash {
				// Convert byte array to comma-separated string
				var taskStr string
				for i, b := range task {
					if i > 0 {
						taskStr += ","
					}
					taskStr += fmt.Sprintf("%d", b)
				}

				output := fmt.Sprintf("Match found: %s, Hash Name: %s, Byte Array: %s\n", taskStr, hashName, hex.EncodeToString(task))
				fmt.Print(output)
				buffer = append(buffer, output...)
				if len(buffer) >= 4096 {
					if _, err := file.Write(buffer); err != nil {
						fmt.Printf("Error writing to file: %v\n", err)
					}
					buffer = buffer[:0]
				}
			}
		}
	}

	// Write any remaining data in the buffer
	if len(buffer) > 0 {
		if _, err := file.Write(buffer); err != nil {
			fmt.Printf("Error writing to file: %v\n", err)
		}
	}
}

func generateHashes(data []byte) map[string]string {
	hashes := make(map[string]string)

	// SHA-512
	sha512Hash := sha512.Sum512(data)
	hashes["SHA-512"] = hex.EncodeToString(sha512Hash[:])

	// SHA3-512
	sha3Hash := sha3.Sum512(data)
	hashes["SHA3-512"] = hex.EncodeToString(sha3Hash[:])

	// Blake2b-512
	blake2bHash := blake2b.Sum512(data)
	hashes["Blake2b-512"] = hex.EncodeToString(blake2bHash[:])

	return hashes
}

func main() {
	length := 50
	if len(os.Args) > 1 {
		length, _ = strconv.Atoi(os.Args[1])
	}
	program := NewProgram()

	// Read the existing hash from file
	existingHashBytes, err := ioutil.ReadFile("existinghash.txt")
	if err != nil {
		fmt.Printf("Error reading existing hash: %v\n", err)
		return
	}
	existingHash := string(existingHashBytes)

	var wg sync.WaitGroup
	numWorkers := 10
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go processTasks(program.tasks, &wg, existingHash)
	}

	program.GenerateAllByteArrays(length)
	wg.Wait()
}
