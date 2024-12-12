package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"
)

type LanguageStas struct {
	FileCount int
	LineCount int
	ByteCount int64
	mutex     sync.Mutex
}

type FileResult struct {
	path     string
	language string
}

var languageExtMap = map[string]string{
	".go":    "Go",
	".py":    "Python",
	".js":    "JavaScript",
	".ts":    "TypeScript",
	".java":  "Java",
	".cpp":   "C++",
	".c":     "C",
	".rb":    "Ruby",
	".php":   "PHP",
	".rs":    "Rust",
	".swift": "Swift",
	".kt":    "Kotlin",
}

func main() {
	startTime := time.Now()

	excludePtr := flag.String("exclude", "", "Comma-separated list of file pattern to exclude (eg. '*.json, *.yml')")
	flag.Parse()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory : %v\n", err)
		os.Exit(1)
	}
	desktopPath := filepath.Join(homeDir, "Desktop")

	// process exclude patterns
	excludePatterns := strings.Split(*excludePtr, ",")
	if *excludePtr == "" {
		excludePatterns = nil
	}

	stats := make(map[string]*LanguageStas)
	var statsMutex sync.Mutex

	// creating channels for the pipelines
	filesChan := make(chan FileResult, 1000)
	done := make(chan bool)
	numWorkers := runtime.NumCPU()

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for result := range filesChan {
				processFile(result.path, result.language, stats, &statsMutex)
			}
		}()
	}

	// file discovery goroutine
	go func() {
		err := filepath.Walk(desktopPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			for _, pattern := range excludePatterns {
				matched, err := filepath.Match(strings.TrimSpace(pattern), filepath.Base(path))
				if err != nil || matched {
					return nil
				}
			}

			ext := strings.ToLower(filepath.Ext(path))
			if lang, ok := languageExtMap[ext]; ok {
				filesChan <- FileResult{path: path, language: lang}
			}

			return nil
		})
		if err != nil {
			fmt.Printf("Error walking directort : %v\n", err)
		}

		close(filesChan)
	}()

	// waiting for all the workers to complete their allocated work
	wg.Wait()
	close(done)

	languages := make([]string, 0, len(stats))
	for lang := range stats {
		languages = append(languages, lang)
	}
	sort.Strings(languages)

	// tablewriter package
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmt.Fprintf(w, "\n Code Report \n \n")
	fmt.Fprintf(w, "Language\tFiles\tLines\tSize (KB)\t\n")
	fmt.Fprintf(w, "--------\t-----\t-----\t---------\t\n")

	totalFiles := 0
	totalLines := 0
	totalSize := int64(0)

	for _, lang := range languages {
		stat := stats[lang]
		totalFiles += stat.FileCount
		totalLines += stat.LineCount
		totalSize += stat.ByteCount
		fmt.Fprintf(w, "%s\t%d\t%d\t%.2f\t\n", lang, stat.FileCount, stat.LineCount, float64(stat.ByteCount)/1024) // dividing by 1024 to get the KB.
	}

	fmt.Fprintf(w, "--------\t-----\t-----\t---------\t\n")
	fmt.Fprintf(w, "Total\t%d\t%d\t%.2f\t\n", totalFiles, totalLines, float64(totalSize)/1024)
	w.Flush()

	fmt.Printf("\n Execution Time : %.2f seconds\n", time.Since(startTime).Seconds())
	if len(excludePatterns) > 0 {
		fmt.Println("\n Excluded Patterns :")
		for _, pattern := range excludePatterns {
			if pattern != ""{
				fmt.Printf("   • %s\n", strings.TrimSpace(pattern))
			}
		}
	}
}

func processFile(path, language string, stats map[string]*LanguageStas, statsMutex *sync.Mutex) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan() {
		lineCount++
	}

	statsMutex.Lock()
	if _, exists := stats[language]; !exists {
		stats[language] = &LanguageStas{}
	}
	stats[language].FileCount++
	stats[language].LineCount += lineCount
	stats[language].ByteCount += info.Size()
	statsMutex.Unlock()
}
