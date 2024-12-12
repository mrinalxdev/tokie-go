package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type LanguageStas struct {
	FileCount int
	LineCount int
	mutex sync.Mutex
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

func main(){
	excludePtr := flag.String("exclude", "", "Comma-separated list of file pattern to exclude (eg. '*.json, *.yml')")
	flag.Parse()

	excludePatterns := []string{}
	if *excludePtr != "" {
		excludePatterns = strings.Split(*excludePtr, ",")
		for i, pattern := range excludePatterns {
			excludePatterns[i] = strings.TrimSpace(pattern)
		}
	}

	stats := make(map[string]*LanguageStas)
	var statsMutex sync.Mutex

	startDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory : %v\n", err)
		os.Exit(1)
	}

	// waitgroups for goroutines
	var wg sync.WaitGroup

	// process files
	err = filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir(){
			return nil
		}

		// checking for excluded file flags to know it should be skipped 
		for _, pattern := range excludePatterns {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				fmt.Println("Error matching pattern %s: %v\n", pattern, err)
				continue
			}
			if matched {
				return nil
			}
		}

		ext := strings.ToLower(filepath.Ext(path))
		if lang, ok := languageExtMap[ext]; ok {
			wg.Add(1)
			go func(filePath, language string){
				defer wg.Done()

				// counting the line in the file for the respective the code
				// todo : will be adding logic to ignore the commencts
				lineCount, err := countLines(filePath)
				if err != nil {
					fmt.Println("Error counting lines in %s: %v\n", filePath, err)
					return
				}

				statsMutex.Lock()
				if _, exists := stats[language]; !exists {
					stats[language] = &LanguageStas{}
				}
				stats[language].mutex.Lock()
				stats[language].FileCount++
				stats[language].LineCount += lineCount
				stats[language].mutex.Unlock()
				statsMutex.Unlock()
			}(path, lang)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking directory : %v\n", err)
		os.Exit(1)
	}


	// goroutines to end
	wg.Wait()
	fmt.Println("\n Code stats")
	totalFiles := 0
	totalLines := 0

	for lang, stat := range stats {
		totalFiles += stat.FileCount
		totalLines += stat.LineCount
		fmt.Printf("\n %:\n", lang)
		fmt.Printf(" Files : %d\n", stat.FileCount)
		fmt.Printf(" Lines of Code : %d\n", stat.LineCount)
	}

	fmt.Println("\n Summary")
	fmt.Println("=============")
	fmt.Printf("Total Files Scanned : %d\n", totalFiles)
	fmt.Printf("Total Lines of Code : %d\n", totalLines)

	if len(excludePatterns) > 0 {
		fmt.Println("\n Excluded Patterns:")
		for _, pattern := range excludePatterns {
			fmt.Printf("    -> %s\n", pattern)
		}
	}
}

func countLines(filePath string) (int, error){
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := 0
	for scanner.Scan(){
		lineCount++
	}

	return lineCount, scanner.Err()
}