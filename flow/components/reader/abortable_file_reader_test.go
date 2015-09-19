package reader

import (
	"os"
	"syscall"
	"testing"

	"fmt"
	"log"
	"time"

	"sync"

	"github.com/stretchr/testify/assert"
)

func TestNewInstanceHasFileReader(t *testing.T) {
	assert := assert.New(t)

	// given
	file := new(os.File)
	c := newFileReaderMockedConfiguration(file)

	// when
	sut := NewAbortableFileReader(c)

	// then
	assert.NotNil(sut.fileReader, "File reader should not be empty.")
}

func TestPipeIncludingFileReader(t *testing.T) {
	assert := assert.New(t)

	// given
	file, error := openFuzzSetFile("fuzz_01.txt")
	assert.Nil(error, "Fuzz set file must be available, error: %v", error)

	c := newFileReaderMockedConfiguration(file)
	sut := NewAbortableFileReader(c)

	var out = make(chan string, 3) // number of lines in fuzz file

	// when
	sut.Pipe(out)

	// then
	assert.Equal("001", <-out, "Invalid line read from file.")
	assert.Equal("002", <-out, "Invalid line read from file.")
	assert.Equal("003", <-out, "Invalid line read from file.")
	assert.Len(out, 0, "Out buffer should be empty now.")

	assert.Equal(^(uintptr(0)), file.Fd(), "File descriptor should be closed now.")
}

func TestAbortedPiping(t *testing.T) {
	assert := assert.New(t)

	// given
	sut := AbortableFileReader{}

	const numberOfStrings = 20
	var in = make(chan string, numberOfStrings+numberOfStrings)
	var done = make(chan bool, 1)
	var out = make(chan string, numberOfStrings+numberOfStrings)
	writeStrings(in, 1, numberOfStrings)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// when
		sut.doPiping(in, out, done)
	}()

	waitForStringsToBePiped(in, out, numberOfStrings)

	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	wg.Wait()

	// then
	for i := 1; i <= numberOfStrings; i++ {
		expected := fmt.Sprintf("%03d", i)
		actual := <-out
		assert.Equal(expected, actual, "Expected value is different than actual.")
	}

	assert.Len(out, 0, "Out buffer should be empty right now.")
	assert.Len(done, 0, "No done event should appear.")
}

func writeStrings(in chan string, from int, to int) {
	for i := from; i <= to; i++ {
		in <- fmt.Sprintf("%03d", i)
	}
}

func waitForStringsToBePiped(in chan string, out chan string, numberOfStrings int) {
	for len(out) != numberOfStrings {
		log.Printf("Waiting for all strings to be piped (got: %d/%d).\n", len(out), numberOfStrings)
		time.Sleep(10 * time.Millisecond)
	}
	log.Printf("To output channel there has been piped: %d/%d strings.\n", len(out), numberOfStrings)
}
