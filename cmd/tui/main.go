package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/shr-go/bili_live_tui/internal/tui"
	"github.com/shr-go/bili_live_tui/pkg/logging"
	"os"
)

func main() {
	logging.Infof("tui start")
	client := tui.GetCustomHttpClient()
	room, err := tui.PrepareEnterRoom(client)
	if err != nil || room == nil {
		logging.Fatalf("Connect server error, err=%v", err)
	}
	p := tea.NewProgram(tui.InitialModel(room), tea.WithAltScreen(), tea.WithMouseCellMotion())
	go tui.ReceiveMsg(p, room)
	go tui.PoolWindowSize(p)
	if err := p.Start(); err != nil {
		logging.Fatalf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
