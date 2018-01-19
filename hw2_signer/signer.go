package main

import (
	"fmt"
	"sync"
	"sort"
	"strings"
	"strconv"
)

const multiHashSubHashes = 6

type HashNode struct {
	id int
	value string
}

func SingleHash(in, out chan interface{}) {
	var wg sync.WaitGroup
	singleHashCh := make(chan string)

	for input := range in {
		wg.Add(1)

		inputAsString := strconv.Itoa(convertToInt(input))
		md5 := DataSignerMd5(inputAsString)

		go calculateSingleHashSum(&wg, singleHashCh, inputAsString, md5)
	}

	go func(wg *sync.WaitGroup, singleHashCh chan string) {
		wg.Wait()
		close(singleHashCh)
	}(&wg, singleHashCh)

	for singleHash := range singleHashCh {
		out <- singleHash
	}
}

// Calculate sum of hashes for SingleHash
func calculateSingleHashSum(wg *sync.WaitGroup, out chan string, inputAsString string, inputAsMd5 string) {
	hashFromInputCh := make(chan string)
	hashFromMd5Ch := make(chan string)

	go func(out chan string, input string) {
		out <- DataSignerCrc32(input)
	}(hashFromInputCh, inputAsString)

	go func(out chan string, input string) {
		out <- DataSignerCrc32(input)
	}(hashFromMd5Ch, inputAsMd5)

	hashFromInput := <- hashFromInputCh
	hashFromMd5 := <- hashFromMd5Ch

	out <- fmt.Sprintf("%v~%v", hashFromInput, hashFromMd5)
	wg.Done()
}

func MultiHash(in, out chan interface{}) {
	var wgOuter sync.WaitGroup
	multiHashOuterCh := make(chan string)

	for input := range in {
		var wgInner sync.WaitGroup
		inputAsString := convertToString(input)
		multiHashInnerCh := make(chan HashNode)

		wgOuter.Add(1)
		wgInner.Add(multiHashSubHashes)
		for i := 0; i < multiHashSubHashes; i++ {
			go calculateInnerMultiHash(&wgInner, i, inputAsString, multiHashInnerCh)
		}
		go func(wgInner *sync.WaitGroup, multiHashInnerCh chan HashNode) {
			wgInner.Wait()
			close(multiHashInnerCh)
		}(&wgInner, multiHashInnerCh)

		go calculateOuterMultiHash(&wgOuter, multiHashInnerCh, multiHashOuterCh)
	}

	go func(wgOuter *sync.WaitGroup, multiHashCh chan string) {
		wgOuter.Wait()
		close(multiHashCh)
	}(&wgOuter, multiHashOuterCh)

	for multiHash := range multiHashOuterCh {
		out <- multiHash
	}
}

// Calculate multi hash for inner loop
func calculateInnerMultiHash(wg *sync.WaitGroup, position int, input string, out chan HashNode) {
	out <- HashNode{position, DataSignerCrc32(fmt.Sprintf("%v%v", position, input))}
	wg.Done()
}

// Calculate multi hash for outer loop
func calculateOuterMultiHash(wg *sync.WaitGroup, in chan HashNode, out chan string) {
	hashNodes := map[int]string{}
	var hashNodeKeys []int

	for o := range in {
		hashNodes[o.id] = o.value
		hashNodeKeys = append(hashNodeKeys, o.id)
	}
	sort.Ints(hashNodeKeys)

	var results []string
	for i := range hashNodeKeys {
		results = append(results, hashNodes[i])
	}

	out <- strings.Join(results, "")
	wg.Done()
}

func CombineResults(in, out chan interface{}) {
	var results []string

	for input := range in {
		inputAsString := convertToString(input)
		results = append(results, inputAsString)
	}

	sort.Strings(results)
	out <- strings.Join(results, "_")
}

// Convert interface{} to string
func convertToString(input interface{}) string {
	inputAsString, ok := input.(string)
	if !ok {
		fmt.Errorf("can't convert %T to string", input)
	}

	return inputAsString
}

// Convert interface{} to int
func convertToInt(input interface{}) int {
	inputAsInt, ok := input.(int)
	if !ok {
		fmt.Errorf("can't convert %T to int", input)
	}

	return inputAsInt
}

func ExecutePipeline(jobs ...job) {
	var wg sync.WaitGroup
	in := make(chan interface{})
	out := make(chan interface{})

	wg.Add(len(jobs))
	for _, job := range jobs {
		go runJob(&wg, job, in, out)

		// Make `out` channel as `in` channel for next job
		in = out
		out = make(chan interface{})
	}
	wg.Wait()
	close(out)
}

// Run a specific job
func runJob(wg *sync.WaitGroup, job job, in, out chan interface{}) {
	job(in, out)

	wg.Done()
	close(out)
}
