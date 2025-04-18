package live_room

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"

	"github.com/shr-go/bili_live_tui/api"
	"github.com/shr-go/bili_live_tui/pkg/logging"
)

// LiveRoom 直播间控制器，用于管理WebSocket连接和弹幕信息
type LiveRoom struct {
	roomID            uint64
	liveRoom          *api.LiveRoom
	client            *http.Client
	danmuSubscribers  map[chan *api.DanmuMessage]struct{}
	subscribersMutex  sync.Mutex
	roomInfo          *RoomInfo
	roomInfoMutex     sync.Mutex
	connected         bool
	isReconnecting    bool
	reconnectAttempts int
}

// RoomInfo 直播间信息
type RoomInfo struct {
	Title        string `json:"title"`
	StreamerName string `json:"streamerName"`
	Online       int    `json:"onlineCount"`
	Popularity   int    `json:"popularity"`
	Likes        int    `json:"likes"`
}

// NewLiveRoom 创建一个新的直播间控制器
func NewLiveRoom(roomID uint64) *LiveRoom {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:     jar,
		Timeout: 10 * time.Second,
	}

	loadCookies(client, "COOKIE.DAT")

	return &LiveRoom{
		roomID:           roomID,
		client:           client,
		danmuSubscribers: make(map[chan *api.DanmuMessage]struct{}),
		roomInfo: &RoomInfo{
			Title:        "加载中...",
			StreamerName: "加载中...",
			Online:       0,
			Popularity:   0,
			Likes:        0,
		},
	}
}

// Start 开始连接直播间
func (lr *LiveRoom) Start() error {
	logging.Infof("Connecting to live room: %d", lr.roomID)

	var err error
	lr.liveRoom, err = AuthAndConnect(lr.client, lr.roomID)
	if err != nil {
		return fmt.Errorf("连接直播间失败: %v", err)
	}

	// 更新直播间信息
	roomInfo, err := GetRoomInfo(lr.client, lr.roomID)
	if err == nil {
		lr.roomInfoMutex.Lock()
		lr.roomInfo.Title = roomInfo.Data.Title
		streamerInfo := GetStreamerInfo(lr.client, uint64(roomInfo.Data.Uid))
		if streamerInfo != nil {
			lr.roomInfo.StreamerName = streamerInfo.Data.Name
		}
		lr.roomInfo.Online = roomInfo.Data.Online
		lr.roomInfoMutex.Unlock()
	}

	// 处理弹幕消息
	go lr.handleDanmuMessages()

	lr.connected = true
	return nil
}

// Stop 停止连接
func (lr *LiveRoom) Stop() {
	if lr.liveRoom != nil {
		close(lr.liveRoom.DoneChan)
	}
	lr.connected = false

	// 通知所有订阅者连接已关闭
	lr.subscribersMutex.Lock()
	defer lr.subscribersMutex.Unlock()

	for ch := range lr.danmuSubscribers {
		close(ch)
	}
	lr.danmuSubscribers = make(map[chan *api.DanmuMessage]struct{})
}

// SubscribeDanmu 订阅弹幕消息
func (lr *LiveRoom) SubscribeDanmu() chan *api.DanmuMessage {
	lr.subscribersMutex.Lock()
	defer lr.subscribersMutex.Unlock()

	ch := make(chan *api.DanmuMessage, 50)
	lr.danmuSubscribers[ch] = struct{}{}
	return ch
}

// UnsubscribeDanmu 取消订阅弹幕消息
func (lr *LiveRoom) UnsubscribeDanmu(ch chan *api.DanmuMessage) {
	lr.subscribersMutex.Lock()
	defer lr.subscribersMutex.Unlock()

	if _, ok := lr.danmuSubscribers[ch]; ok {
		delete(lr.danmuSubscribers, ch)
		close(ch)
	}
}

// handleDanmuMessages 处理弹幕消息
func (lr *LiveRoom) handleDanmuMessages() {
	for msg := range lr.liveRoom.MessageChan {
		// 将消息广播给所有订阅者
		lr.subscribersMutex.Lock()
		for ch := range lr.danmuSubscribers {
			select {
			case ch <- msg:
				// 消息发送成功
			default:
				// 通道已满，丢弃消息
				logging.Warnf("Subscriber channel is full, dropping message")
			}
		}
		lr.subscribersMutex.Unlock()

		// 处理特殊消息类型，例如人气值更新
		if msg.Cmd == "WATCHED_CHANGE" {
			if data, ok := msg.Data["num"].(float64); ok {
				lr.roomInfoMutex.Lock()
				lr.roomInfo.Online = int(data)
				lr.roomInfoMutex.Unlock()
			}
		} else if msg.Cmd == "LIKE_INFO_V3_UPDATE" {
			if data, ok := msg.Data["click_count"].(float64); ok {
				lr.roomInfoMutex.Lock()
				lr.roomInfo.Likes = int(data)
				lr.roomInfoMutex.Unlock()
			}
		}
	}

	// 如果消息通道关闭且未主动停止，尝试重连
	if lr.connected && !lr.isReconnecting {
		lr.attemptReconnect()
	}
}

// attemptReconnect 尝试重新连接
func (lr *LiveRoom) attemptReconnect() {
	lr.isReconnecting = true
	defer func() { lr.isReconnecting = false }()

	maxAttempts := 5
	for lr.reconnectAttempts < maxAttempts {
		lr.reconnectAttempts++
		logging.Infof("尝试重新连接直播间: %d (第 %d 次尝试)", lr.roomID, lr.reconnectAttempts)

		// 等待一段时间后重试
		time.Sleep(time.Duration(lr.reconnectAttempts) * 2 * time.Second)

		// 重新连接
		var err error
		lr.liveRoom, err = AuthAndConnect(lr.client, lr.roomID)
		if err == nil {
			lr.reconnectAttempts = 0
			go lr.handleDanmuMessages()
			logging.Infof("重新连接成功")
			return
		}

		logging.Errorf("重新连接失败: %v", err)
	}

	// 超过最大重试次数
	logging.Errorf("重新连接失败，已达到最大重试次数")
}

// SendDanmaku 发送弹幕
func (lr *LiveRoom) SendDanmaku(message string) error {
	if lr.liveRoom == nil || lr.liveRoom.RoomUserInfo == nil {
		return fmt.Errorf("未登录或直播间未连接")
	}

	return SendDanmu(
		lr.client,
		lr.liveRoom.RoomID,
		message,
		lr.liveRoom.RoomUserInfo.Danmu.Color,
		lr.liveRoom.RoomUserInfo.Danmu.Mode,
		lr.liveRoom.CSRF,
	)
}

// GetRoomInfo 获取直播间信息
func (lr *LiveRoom) GetRoomInfo() map[string]interface{} {
	lr.roomInfoMutex.Lock()
	defer lr.roomInfoMutex.Unlock()

	return map[string]interface{}{
		"title":        lr.roomInfo.Title,
		"streamerName": lr.roomInfo.StreamerName,
		"onlineCount":  lr.roomInfo.Online,
		"popularity":   lr.roomInfo.Popularity,
		"likes":        lr.roomInfo.Likes,
	}
}

// loadCookies 从文件加载cookie
func loadCookies(client *http.Client, path string) {
	// 此处省略具体实现，实际项目中需要从文件中加载cookie
	// 由于这部分功能复杂，这里只是一个占位符
	// 真实实现应该读取cookie文件并设置到client.Jar中
	logging.Debugf("Loading cookies from file: %s", path)
}

// GetStreamerInfo 获取主播信息
func GetStreamerInfo(client *http.Client, uid uint64) *struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Name string `json:"name"`
	} `json:"data"`
} {
	// 此处省略具体实现，实际项目中需要调用API获取主播信息
	return &struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Name string `json:"name"`
		} `json:"data"`
	}{
		Code: 0,
		Data: struct {
			Name string `json:"name"`
		}{
			Name: "主播",
		},
	}
}

// SendDanmu 发送弹幕
func SendDanmu(client *http.Client, roomID uint64, message string, color, mode int, csrf string) error {
	// 此处省略具体实现，实际项目中需要调用API发送弹幕
	logging.Infof("Sending danmu: %s", message)
	return nil
}
