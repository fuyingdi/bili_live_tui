package api

import (
	"os"

	"github.com/BurntSushi/toml"
)

type BiliLiveConfig struct {
	RoomID         uint64 `toml:"room_id"`
	ChatBuffer     int    `toml:"chat_buffer"`
	ShowShipLevel  bool   `toml:"show_ship_level"`
	ShowMedalName  bool   `toml:"show_medal_name"`
	ShowMedalLevel bool   `toml:"show_medal_level"`
	ColorMode      bool   `toml:"color_mode"`
	ShowRoomTitle  bool   `toml:"show_room_title"`
	ShowRoomNumber bool   `toml:"show_room_number"`
	ShowTimestamp  bool   `toml:"show_timestamp"`
}

// LoadConfig 从指定路径加载配置文件
func LoadConfig(path string) (*BiliLiveConfig, error) {
	config := &BiliLiveConfig{
		RoomID:         0,
		ChatBuffer:     200,
		ShowShipLevel:  true,
		ShowMedalName:  true,
		ShowMedalLevel: true,
		ColorMode:      true,
		ShowRoomTitle:  true,
		ShowRoomNumber: true,
		ShowTimestamp:  true,
	}

	// 检查配置文件是否存在
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		// 配置文件不存在，创建默认配置
		file, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		encoder := toml.NewEncoder(file)
		err = encoder.Encode(config)
		if err != nil {
			return nil, err
		}
		return config, nil
	}

	// 配置文件存在，加载配置
	_, err = toml.DecodeFile(path, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
