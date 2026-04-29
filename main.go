package main

import (
	"fmt"
	"os"

	roulette "github.com/ramonmacias/tui-roulette/internal"

	tea "charm.land/bubbletea/v2"
)

func main() {
	storage, err := roulette.NewFileStorage()
	if err != nil {
		fmt.Printf("Failed to initialize storage: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(roulette.InitialModel(storage))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
