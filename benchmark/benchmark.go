package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"net/http"
	_ "net/http/pprof"

	ac "github.com/codefever/ahocorasick"
)

// flags
var (
	flagDict        = flag.String("dict", "./cn/dictionary.txt", "Path for dictionary")
	flagText        = flag.String("text", "./cn/text.txt", "Path for text")
	flagCPUProfile  = flag.String("cpuprofile", "", "write cpu profile to `file`")
	flagMemProfile  = flag.String("memprofile", "", "write memory profile to `file`")
	flagRunnerType  = flag.String("runner", "AC", "Runner type: AC/Dummy")
	flagPrintResult = flag.Bool("printResult", false, "Print result line by line")
)

type testRunner interface {
	Init(dict []string)
	Run(text string) []interface{}
	Name() string
}

type acRunner struct {
	searcher *ac.Searcher
}

func (r *acRunner) Init(dict []string) {
	builder := ac.NewBuilder()
	for i, w := range dict {
		builder.Add(w, i)
	}
	r.searcher = builder.Build()
}

func (r *acRunner) Run(text string) []interface{} {
	return r.searcher.Cover(text)
}

func (r *acRunner) Name() string {
	return "AC"
}

type dummyRunner struct {
	dict []string
}

func (r *dummyRunner) Init(dict []string) {
	r.dict = dict
}

func (r *dummyRunner) Run(text string) []interface{} {
	ret := make([]interface{}, 0)
	for i, w := range r.dict {
		if strings.Index(text, w) >= 0 {
			ret = append(ret, i)
		}
	}
	return ret
}

func (r *dummyRunner) Name() string {
	return "Dummy"
}

func getMemAlloc() uint64 {
	mem := new(runtime.MemStats)
	runtime.GC()
	runtime.ReadMemStats(mem)
	return mem.HeapAlloc
}

func readDict(filePath string) ([]string, error) {
	fp, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	ret := make([]string, 0)
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 0 {
			ret = append(ret, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return ret, nil
}

func readText(filePath string) (string, error) {
	fp, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return "", err
	}
	defer fp.Close()

	bytes, err := ioutil.ReadAll(fp)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func runTest(r testRunner, dict []string, text string) []interface{} {
	// init
	memBefore := getMemAlloc()
	timeBefore := time.Now()
	r.Init(dict)
	buildTimeCost := time.Since(timeBefore)
	memAfter := getMemAlloc()
	fmt.Printf("Build[%v]: mem=%v timecost=%v\n", r.Name(), memAfter-memBefore, buildTimeCost)

	// run
	timeBefore = time.Now()
	ret := r.Run(text)
	searchTimeCost := time.Since(timeBefore)
	fmt.Printf("Search[%v]: timecost=%v\n", r.Name(), searchTimeCost)

	return ret
}

func main() {
	flag.Parse()

	dict, err := readDict(*flagDict)
	if err != nil {
		panic(err)
	}

	text, err := readText(*flagText)
	if err != nil {
		panic(err)
	}

	var r testRunner
	fmt.Println("Runner:", *flagRunnerType)
	if *flagRunnerType == "AC" {
		r = &acRunner{}
	} else if *flagRunnerType == "Dummy" {
		r = &dummyRunner{}
	} else {
		panic("What runner type?")
	}

	if *flagCPUProfile != "" {
		f, err := os.Create(*flagCPUProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()

		go func() {
			http.ListenAndServe(":18000", http.DefaultServeMux)
		}()
	}

	ret := runTest(r, dict, text)

	if *flagMemProfile != "" {
		f, err := os.Create(*flagMemProfile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}

	if *flagPrintResult {
		for _, v := range ret {
			i := v.(int)
			fmt.Println(dict[i])
		}
	}
}
