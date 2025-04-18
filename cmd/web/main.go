package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/shr-go/bili_live_tui/api"
	"github.com/shr-go/bili_live_tui/internal/live_room"
	"github.com/shr-go/bili_live_tui/pkg/logging"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有跨域请求，生产环境应该更严格
	},
}

func main() {
	// 初始化日志系统
	logging.InitLogConfig()

	logging.Infof("Starting Bilibili Live Web Server")

	// 初始化直播间
	// 从配置文件读取直播间ID
	config, err := api.LoadConfig("config.toml")
	if err != nil {
		logging.Errorf("Failed to load config: %v", err)
		return
	}

	// 创建直播间实例
	liveRoom := live_room.NewLiveRoom(config.RoomID)

	// 启动直播间连接
	if err := liveRoom.Start(); err != nil {
		logging.Errorf("Failed to start live room: %v", err)
		return
	}
	defer liveRoom.Stop()

	r := gin.Default()

	// 静态文件服务
	r.Static("/static", "./frontend")
	r.StaticFile("/", "./frontend/index.html")

	// API端点
	// 发送弹幕
	r.POST("/api/send", func(c *gin.Context) {
		var req struct {
			Message string `json:"message"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 发送弹幕
		if err := liveRoom.SendDanmaku(req.Message); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		logging.Infof("Sent danmu: %s", req.Message)
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// 获取直播间信息
	r.GET("/api/room_info", func(c *gin.Context) {
		info := liveRoom.GetRoomInfo()
		c.JSON(http.StatusOK, info)
	})

	// WebSocket端点
	r.GET("/ws", func(c *gin.Context) {
		wsHandler(c.Writer, c.Request, liveRoom)
	})

	// 启动服务器
	logging.Infof("Server running at http://localhost:8080")
	r.Run(":8080")
}

// WebSocket处理函数
func wsHandler(w http.ResponseWriter, r *http.Request, liveRoom *live_room.LiveRoom) {
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Errorf("Failed to upgrade to websocket: %v", err)
		return
	}
	defer conn.Close()

	logging.Infof("New WebSocket connection established")

	// 创建一个通道接收弹幕消息
	danmuChan := liveRoom.SubscribeDanmu()
	defer liveRoom.UnsubscribeDanmu(danmuChan)

	// 发送初始房间信息
	initialInfo := map[string]interface{}{
		"type": "room_info",
		"data": liveRoom.GetRoomInfo(),
	}
	if err := conn.WriteJSON(initialInfo); err != nil {
		logging.Errorf("Failed to send initial room info: %v", err)
		return
	}
	logging.Infof("Sent initial room info to client")

	// 监听客户端消息
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				logging.Infof("WebSocket connection closed: %v", err)
				return
			}
		}
	}()

	// 发送弹幕消息给客户端
	for danmu := range danmuChan {
		// 记录原始弹幕信息，以便调试
		logging.Debugf("Raw danmu message: Cmd=%s, Info len=%d", danmu.Cmd, len(danmu.Info))
		if len(danmu.Info) > 0 {
			logging.Debugf("Info[0] type: %T", danmu.Info[0])
		}

		userName := getDanmuUserName(danmu)
		content := getDanmuContent(danmu)
		userLevel := getDanmuUserLevel(danmu)

		// 如果解析出的内容为空，尝试使用备用方法解析
		if content == "" || userName == "用户" {
			userName, content = tryAlternativeParser(danmu)
			logging.Debugf("Using alternative parser: userName=%s, content=%s", userName, content)
		}

		message := map[string]interface{}{
			"type": "danmu",
			"data": map[string]interface{}{
				"userName":  userName,
				"content":   content,
				"userLevel": userLevel,
			},
		}

		if err := conn.WriteJSON(message); err != nil {
			logging.Errorf("Failed to send danmu: %v", err)
			return
		}
		logging.Debugf("Sent danmu to client: %s from %s", content, userName)
	}
}

// 从弹幕消息中获取用户名 - 标准方法
func getDanmuUserName(danmu *api.DanmuMessage) string {
	// 检查新格式（data字段中的userName）
	if danmu.Data != nil {
		if userName, ok := danmu.Data["userName"].(string); ok && userName != "" {
			return userName
		}
	}

	// 旧格式（Info数组）
	if danmu.Cmd == "DANMU_MSG" && len(danmu.Info) > 2 {
		if userInfo, ok := danmu.Info[2].([]interface{}); ok && len(userInfo) > 0 {
			if userName, ok := userInfo[0].(string); ok {
				return userName
			}
		}
	}
	return "用户"
}

// 从弹幕消息中获取内容 - 标准方法
func getDanmuContent(danmu *api.DanmuMessage) string {
	// 检查新格式（data字段中的content）
	if danmu.Data != nil {
		if content, ok := danmu.Data["content"].(string); ok && content != "" {
			return content
		}
	}

	// 旧格式（Info数组）
	if danmu.Cmd == "DANMU_MSG" && len(danmu.Info) > 1 {
		if content, ok := danmu.Info[1].(string); ok {
			return content
		}
	}
	return ""
}

// 从弹幕消息中获取用户等级 - 标准方法
func getDanmuUserLevel(danmu *api.DanmuMessage) int {
	// 检查新格式（data字段中的userLevel）
	if danmu.Data != nil {
		if level, ok := danmu.Data["userLevel"].(float64); ok {
			return int(level)
		}
	}

	// 旧格式（Info数组）
	if danmu.Cmd == "DANMU_MSG" && len(danmu.Info) > 4 {
		if userInfo, ok := danmu.Info[4].([]interface{}); ok && len(userInfo) > 0 {
			if level, ok := userInfo[0].(float64); ok {
				return int(level)
			}
		}
	}
	return 0
}

// 尝试使用备用方法解析弹幕
func tryAlternativeParser(danmu *api.DanmuMessage) (userName string, content string) {
	userName = "用户"
	content = ""

	// 检查是否是新版DANMU_MSG格式
	if danmu.Cmd == "DANMU_MSG" {
		if len(danmu.Info) > 0 {
			// 尝试从不同位置获取弹幕内容
			if len(danmu.Info) > 1 {
				// 标准位置 (位置1)
				if str, ok := danmu.Info[1].(string); ok && str != "" {
					content = str
				}
			}

			// 尝试从不同位置获取用户名
			if len(danmu.Info) > 2 {
				// 方法1: 如果Info[2]是一个map
				if userData, ok := danmu.Info[2].(map[string]interface{}); ok {
					if name, ok := userData["uname"].(string); ok && name != "" {
						userName = name
						return
					}
				}

				// 方法2: 如果Info[2]是数组，尝试多种位置
				if userArr, ok := danmu.Info[2].([]interface{}); ok {
					if len(userArr) > 0 {
						if name, ok := userArr[0].(string); ok && name != "" {
							userName = name
							return
						}
					}
					// 尝试更深层次的数组
					if len(userArr) > 1 {
						if subArr, ok := userArr[1].([]interface{}); ok && len(subArr) > 0 {
							if name, ok := subArr[0].(string); ok {
								userName = name
								return
							}
						}
					}
				}
			}

			// 直接寻找字段
			for i := 0; i < len(danmu.Info); i++ {
				// 如果是字符串且看起来像弹幕内容
				if i != 1 { // 已经检查过位置1
					if str, ok := danmu.Info[i].(string); ok && len(str) > 0 && len(str) < 200 {
						if content == "" { // 优先保留已找到的内容
							content = str
						}
					}
				}
			}
		}
	}

	// 尝试从Data字段获取
	if danmu.Data != nil {
		if msgContent, ok := danmu.Data["msg"].(string); ok && msgContent != "" {
			content = msgContent
		}
		if nickname, ok := danmu.Data["nickname"].(string); ok && nickname != "" {
			userName = nickname
		} else if uname, ok := danmu.Data["uname"].(string); ok && uname != "" {
			userName = uname
		}
	}

	// 如果是其他类型的消息，尝试展示命令类型
	if content == "" {
		content = "[" + danmu.Cmd + "]"
	}

	return
}
