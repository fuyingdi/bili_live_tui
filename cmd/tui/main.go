package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/shr-go/bili_live_tui/api"
	"github.com/shr-go/bili_live_tui/internal/tui"
	"github.com/shr-go/bili_live_tui/pkg/logging"
)

func main() {
	logging.Infof("tui start")
	client := api.NewHTTPClient()
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
