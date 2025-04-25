package api

const (
	// DefaultUserAgent 设置一个常见浏览器的 User-Agent
	DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"
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
