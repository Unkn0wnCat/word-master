package main

import (
	"fmt"
	"github.com/c-bata/go-prompt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	wordList []string
)

func handleExit() {
	rawModeOff := exec.Command("/bin/stty", "-raw", "echo")
	rawModeOff.Stdin = os.Stdin
	_ = rawModeOff.Run()
	rawModeOff.Wait()
}

func completer(d prompt.Document) []prompt.Suggest {
	baseCommands := []prompt.Suggest{
		{Text: "load", Description: "Loads a wordlist from file"},
		{Text: "length", Description: "Filters words by length"},
		{Text: "mask", Description: "Masks words"},
		{Text: "!mask", Description: "Masks words which are not it"},
		{Text: "letters", Description: "Only keeps words containing all letters"},
		{Text: "!letters", Description: "Removes words containing any letters"},
		{Text: "print", Description: "Prints the resulting list"},
		{Text: "count", Description: "Shows count of resulting list"},
		{Text: "exit", Description: "Exit app"},
		{Text: "clean", Description: "Clean list"},
	}

	if strings.HasPrefix(d.Text, "load ") {
		path := d.GetWordBeforeCursor()

		if path == "load" {
			return prompt.FilterHasPrefix(baseCommands, d.GetWordBeforeCursor(), true)
		}

		dir := path

		if !strings.HasSuffix(path, "/") {
			parts := strings.Split(path, "/")

			dir = strings.Join(parts[:len(parts)-1], "/")
		}

		if dir == "" {
			dir = "."
		}

		files, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}

		var pathSuggestions []prompt.Suggest

		for _, file := range files {
			desc := "Directory"

			if !file.IsDir() {
				desc = "File"

				info, err := file.Info()
				if err == nil {
					desc = fmt.Sprintf("File (%d kB)", info.Size()/1000)
				}
			}

			pathSuggestions = append(pathSuggestions, prompt.Suggest{
				Text:        file.Name(),
				Description: desc,
			})
		}

		return prompt.FilterHasPrefix(pathSuggestions, d.GetWordBeforeCursor(), true)
	}

	return prompt.FilterHasPrefix(baseCommands, d.GetWordBeforeCursor(), true)
}

func help() {
	fmt.Println(" --[ HELP ]-- ")
	fmt.Println()
	fmt.Println(" load <file>: Loads a wordlist from file")
	fmt.Println(" length <number>: Filters words by length")
	fmt.Println("  mask <mask>: Masks words")
	fmt.Println(" !mask <mask>: Masks words which are not it")
	fmt.Println("  letters <letters>: Only keeps words containing all letters")
	fmt.Println(" !letters <letters>: Removes words containing any letters")
	fmt.Println(" print: Prints the resulting list")
	fmt.Println(" exit: Exit app")
	fmt.Println(" clean: Clean list")

	fmt.Println()
	fmt.Println(" --  Mask  -- ")
	fmt.Println()
	fmt.Println(" A mask is a string with exactly as many letters")
	fmt.Println(" as you are searching for, containing \"-\" for")
	fmt.Println(" unknown letters.")
}

func parseParts(command string) []string {
	var parts []string

	current := ""
	inQuotes := false
	escapeNext := false

	for _, char := range command {
		if char == '\\' && !escapeNext {
			escapeNext = true
			continue
		}

		if char == '"' && !escapeNext {
			inQuotes = !inQuotes
			continue
		}

		if char == ' ' && !inQuotes && !escapeNext {
			parts = append(parts, current)
			current = ""
			continue
		}

		current += string(char)
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func run(command string) {
	if command == "exit" {
		handleExit()
		os.Exit(0)
	}

	parts := parseParts(command)

	if parts[0] == "load" {
		if len(parts) < 2 {
			fmt.Println("Missing argument.")
			return
		}

		for _, part := range parts[1:] {
			load(part)
		}

		printCount()
		return
	}

	if parts[0] == "print" {
		if len(parts) > 1 {
			fmt.Println("Unnecessary Argument.")
			return
		}

		printList()
		return
	}

	if parts[0] == "count" {
		if len(parts) > 1 {
			fmt.Println("Unnecessary Argument.")
			return
		}

		printCount()
		return
	}

	if parts[0] == "length" {
		if len(parts) != 2 && len(parts) != 3 {
			fmt.Println("Invalid Number of Arguments.")
			return
		}

		if len(parts) == 2 {
			length, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Invalid length")
				fmt.Println(err)
				return
			}

			fmt.Printf("Filtering for length %d...\n", length)

			filterByExactLength(length)

			printCount()
			return
		}

		if len(parts) == 3 {
			min, err := strconv.Atoi(parts[1])
			if err != nil {
				fmt.Println("Invalid min")
				fmt.Println(err)
				return
			}

			max, err := strconv.Atoi(parts[2])
			if err != nil {
				fmt.Println("Invalid max")
				fmt.Println(err)
				return
			}

			fmt.Printf("Filtering for %d >= length >= %d...\n", min, max)

			filterByLength(min, max)

			printCount()
			return
		}

		return
	}

	if parts[0] == "!mask" || parts[0] == "mask" {
		if len(parts) < 2 {
			fmt.Println("Missing argument.")
			return
		}

		for _, mask := range parts[1:] {
			if parts[0] == "!mask" {
				filterByMask(mask, true)
				continue
			}

			filterByMask(mask, false)
		}

		printCount()
		return
	}

	if parts[0] == "!letters" || parts[0] == "letters" {
		if len(parts) < 2 {
			fmt.Println("Missing argument.")
			return
		}

		for _, letters := range parts[1:] {
			if parts[0] == "!letters" {
				filterByLetters(letters, true)
				continue
			}

			filterByLetters(letters, false)
		}

		printCount()
		return
	}

	if parts[0] == "clean" {
		if len(parts) != 2 || parts[1] != "yes" {
			fmt.Println("Type \"clean yes\" if you REALLY mean it!")
			return
		}

		wordList = []string{}
		printCount()
		return
	}

	fmt.Println(parts)
}

func filterByLetters(letters string, inverted bool) {
	var newList []string

	for _, word := range wordList {
		wordOk := true

		for _, letter := range letters {
			contains := false

			for _, wordLetter := range word {
				if letter == wordLetter {
					contains = true
					break
				}
			}

			if contains == inverted {
				wordOk = false
				break
			}
		}

		if !wordOk {
			continue
		}

		newList = append(newList, word)
	}

	if sanityCheck(newList) {
		return
	}

	wordList = newList
}

func sanityCheck(list []string) bool {
	if len(list) == 0 {
		fmt.Println("Sanity Check failed. No words would be left, aborting.")
		return true
	}

	return false
}

func filterByMask(mask string, inverted bool) {
	var newList []string

	for _, word := range wordList {
		if len([]rune(mask)) != len([]rune(word)) {
			if inverted {
				newList = append(newList, word)
			}

			continue
		}

		matchedMask := true

		for i, maskChar := range mask {
			if maskChar == '-' {
				continue
			}

			if maskChar != []rune(word)[i] {
				matchedMask = false
				break
			}
		}

		if matchedMask == inverted {
			continue
		}

		newList = append(newList, word)
	}

	if sanityCheck(newList) {
		return
	}

	wordList = newList
}

func filterByExactLength(exact int) {
	var newList []string

	for _, word := range wordList {
		if len(word) != exact {
			continue
		}

		newList = append(newList, word)
	}

	if sanityCheck(newList) {
		return
	}

	wordList = newList
}

func filterByLength(min int, max int) {
	var newList []string

	for _, word := range wordList {
		if len(word) < min || len(word) > max {
			continue
		}

		newList = append(newList, word)
	}

	if sanityCheck(newList) {
		return
	}

	wordList = newList
}

func printCount() {
	fmt.Printf("WordList now contains %d words.\n", len(wordList))
}

func printList() {
	fmt.Println("---- LIST START ----")

	for _, word := range wordList {
		fmt.Println(word)
	}

	fmt.Println("---- LIST   END ----")
}

func main() {
	defer handleExit()

	fmt.Println("Kevin's WordMaster 3000 v1.0")
	fmt.Println()
	t := prompt.New(run, completer, prompt.OptionPrefix("WM] "))
	t.Run()
}

func load(file string) {
	fmt.Printf("Loading \"%s\"...\n", file)

	contentBinary, err := os.ReadFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}

	content := string(contentBinary)

	myList := strings.Split(content, "\n")

	for _, word := range myList {
		word = strings.ToUpper(word)

		word = strings.ReplaceAll(word, "Ä", "AE")
		word = strings.ReplaceAll(word, "Ö", "OE")
		word = strings.ReplaceAll(word, "Ü", "UE")

		wordList = append(wordList, word)
	}

	fmt.Printf("Loaded \"%s\". %d entries added.\n", file, len(myList))
}
