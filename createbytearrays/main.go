package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"sync"
)

type ProcessQueueItem struct {
	HopperString string
}

func GenerateQueueItem(hopperString []byte) ProcessQueueItem {
	return ProcessQueueItem{
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
	sb.WriteString(fmt.Sprintf("%s", item.HopperString))
	return sb.String()
}

type Program struct {
	tasks chan string
}

func NewProgram() *Program {
	return &Program{
		tasks: make(chan string, 1000),
	}
}

func (p *Program) GenerateAllByteArrays(maxArrayLength int) {
	p.GenerateByteArrays(maxArrayLength, 1, nil)
	close(p.tasks)
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

func processTasks(tasks chan string, wg *sync.WaitGroup) {
	defer wg.Done()

	for task := range tasks {
		fmt.Println(task)
	}
}

func main() {
	length := 50
	if len(os.Args) > 1 {
		length, _ = strconv.Atoi(os.Args[1])
	}
	program := NewProgram()

	var wg sync.WaitGroup
	numWorkers := 10
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go processTasks(program.tasks, &wg)
	}

	program.GenerateAllByteArrays(length)
	wg.Wait()
}
