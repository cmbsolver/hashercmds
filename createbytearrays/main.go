package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"sync"

	_ "github.com/lib/pq"
)

type ProcessQueueItem struct {
	Id           string
	HopperString string
}

func GenerateQueueItem(hopperString []byte) ProcessQueueItem {
	id := make([]byte, 16)
	rand.Read(id)
	return ProcessQueueItem{
		Id:           hex.EncodeToString(id),
		HopperString: bytesToCommaSeparatedString(hopperString),
	}
}

func bytesToCommaSeparatedString(b []byte) string {
	var sb bytes.Buffer
	for i, v := range b {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%d", v))
	}
	return sb.String()
}

func (item ProcessQueueItem) GetHopperInsertString() string {
	var sb bytes.Buffer
	sb.WriteString(fmt.Sprintf("INSERT INTO public.\"TB_PROCESS_QUEUE\"(\"ID\", \"HOPPER_STRING\") VALUES ('%s', '%s');\n", item.Id, item.HopperString))
	return sb.String()
}

type Program struct {
	tasks               chan string
	maxCombinations     *big.Int
	currentCombinations *big.Int
}

func NewProgram() *Program {
	return &Program{
		tasks:               make(chan string, 1000),
		maxCombinations:     big.NewInt(0),
		currentCombinations: big.NewInt(0),
	}
}

func (p *Program) GenerateAllByteArrays(maxArrayLength int) {
	p.maxCombinations.Exp(big.NewInt(256), big.NewInt(int64(maxArrayLength)), nil)
	p.currentCombinations.SetInt64(0)
	p.GenerateByteArrays(maxArrayLength, 1, nil)
	close(p.tasks)
	fmt.Println("\nDone generating byte arrays!\n")
}

func (p *Program) GenerateByteArrays(maxArrayLength, currentArrayLevel int, passedArray []byte) {
	if currentArrayLevel == maxArrayLength {
		var wg sync.WaitGroup
		for i := 0; i < 256; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				currentArray := make([]byte, currentArrayLevel)
				if passedArray != nil {
					copy(currentArray, passedArray)
				}
				currentArray[currentArrayLevel-1] = byte(i)
				item := GenerateQueueItem(currentArray)
				p.tasks <- item.GetHopperInsertString()
			}(i)
		}
		wg.Wait()
		p.currentCombinations.Add(p.currentCombinations, big.NewInt(256))
		p.maxCombinations.Sub(p.maxCombinations, big.NewInt(256))

		if new(big.Int).Mod(p.currentCombinations, big.NewInt(256)).Cmp(big.NewInt(0)) == 0 {
			fmt.Printf("Generating %d length byte arrays...\n%s Computed\n%s Remaining\n", maxArrayLength, p.currentCombinations.String(), p.maxCombinations.String())
			os.WriteFile("lasthash.txt", []byte(p.currentCombinations.String()), 0644)
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

func processTasks(tasks chan string, wg *sync.WaitGroup, db *sql.DB) {
	defer wg.Done()

	batchSize := 100
	var batch []string

	for task := range tasks {
		batch = append(batch, task)
		if len(batch) >= batchSize {
			executeBatch(db, batch)
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		executeBatch(db, batch)
	}
}

func executeBatch(db *sql.DB, batch []string) {
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		return
	}

	for _, task := range batch {
		_, err := tx.Exec(task)
		if err != nil {
			log.Printf("Failed to execute task: %v", err)
			tx.Rollback()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
	}
}

func main() {
	length := 50
	if len(os.Args) > 1 {
		length, _ = strconv.Atoi(os.Args[1])
	}
	program := NewProgram()

	connStrBytes, err := ioutil.ReadFile("connStr.txt")
	if err != nil {
		log.Fatalf("Failed to read connection string file: %v", err)
	}
	connStr := string(connStrBytes)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	var wg sync.WaitGroup
	numWorkers := 10
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go processTasks(program.tasks, &wg, db)
	}

	program.GenerateAllByteArrays(length)
	wg.Wait()
}
