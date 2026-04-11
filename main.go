package main

import (
	"fmt"
	"os"
	"roulette/roulette"

	tea "charm.land/bubbletea/v2"
)

func main() {
	p := tea.NewProgram(roulette.InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
