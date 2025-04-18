// 全局变量
let websocket = null;
let reconnectAttempts = 0;
const maxReconnectAttempts = 5;
const reconnectDelay = 3000;

// DOM 元素
const danmuArea = document.getElementById('danmu-area');
const danmuInput = document.getElementById('danmu-input');
const sendBtn = document.getElementById('send-btn');
const roomTitle = document.getElementById('room-title');
const streamerName = document.getElementById('streamer-name');
const onlineCount = document.getElementById('online-count').querySelector('span');
const popularityElement = document.getElementById('popularity');
const likesElement = document.getElementById('likes');

// 初始化函数
function init() {
    connectWebSocket();
    setupEventListeners();
    fetchRoomInfo();
}

// 设置事件监听
function setupEventListeners() {
    // 发送弹幕
    sendBtn.addEventListener('click', sendDanmu);
    danmuInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            sendDanmu();
        }
    });

    // 滚动到最新弹幕
    danmuArea.addEventListener('DOMNodeInserted', () => {
        danmuArea.scrollTop = danmuArea.scrollHeight;
    });
}

// 连接 WebSocket
function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    websocket = new WebSocket(wsUrl);
    
    websocket.onopen = () => {
        console.log('WebSocket 连接已建立');
        reconnectAttempts = 0;
        showSystemMessage('连接成功');
    };
    
    websocket.onmessage = (event) => {
        handleWebSocketMessage(event.data);
    };
    
    websocket.onclose = () => {
        console.log('WebSocket 连接已关闭');
        showSystemMessage('连接已断开');
        
        // 尝试重新连接
        if (reconnectAttempts < maxReconnectAttempts) {
            reconnectAttempts++;
            showSystemMessage(`${reconnectDelay / 1000}秒后尝试重新连接... (${reconnectAttempts}/${maxReconnectAttempts})`);
            setTimeout(connectWebSocket, reconnectDelay);
        } else {
            showSystemMessage('无法连接到服务器，请刷新页面重试');
        }
    };
    
    websocket.onerror = (error) => {
        console.error('WebSocket 错误:', error);
        showSystemMessage('连接出错');
    };
}

// 处理 WebSocket 消息
function handleWebSocketMessage(data) {
    try {
        const message = JSON.parse(data);
        console.log('收到WebSocket消息:', message);
        
        switch (message.type) {
            case 'danmu':
                addDanmuMessage(message.data);
                break;
                
            case 'room_info':
                updateRoomInfo(message.data);
                break;
                
            case 'gift':
                addGiftMessage(message.data);
                break;
                
            case 'super_chat':
                addSuperChatMessage(message.data);
                break;
                
            case 'stats_update':
                updateRoomStats(message.data);
                break;
                
            default:
                console.log('未知消息类型:', message);
        }
    } catch (error) {
        console.error('处理消息出错:', error, '原始数据:', data);
    }
}

// 添加弹幕消息到界面
function addDanmuMessage(data) {
    const danmuItem = document.createElement('div');
    danmuItem.className = 'danmu-item';
    
    const danmuHeader = document.createElement('div');
    danmuHeader.className = 'danmu-header';
    
    const userName = document.createElement('span');
    userName.className = 'user-name';
    userName.textContent = data.userName || '用户';
    danmuHeader.appendChild(userName);
    
    if (data.userLevel) {
        const userLevel = document.createElement('span');
        userLevel.className = 'user-level';
        userLevel.textContent = `LV${data.userLevel}`;
        danmuHeader.appendChild(userLevel);
    }
    
    const danmuContent = document.createElement('div');
    danmuContent.className = 'danmu-content';
    danmuContent.textContent = data.content || '';
    
    danmuItem.appendChild(danmuHeader);
    danmuItem.appendChild(danmuContent);
    
    danmuArea.appendChild(danmuItem);
}

// 添加礼物消息
function addGiftMessage(data) {
    const danmuItem = document.createElement('div');
    danmuItem.className = 'danmu-item';
    
    const danmuHeader = document.createElement('div');
    danmuHeader.className = 'danmu-header';
    
    const userName = document.createElement('span');
    userName.className = 'user-name';
    userName.textContent = data.userName || '用户';
    danmuHeader.appendChild(userName);
    
    const danmuContent = document.createElement('div');
    danmuContent.className = 'danmu-content gift-danmu';
    danmuContent.textContent = `赠送了 ${data.giftName} x${data.giftCount}`;
    
    danmuItem.appendChild(danmuHeader);
    danmuItem.appendChild(danmuContent);
    
    danmuArea.appendChild(danmuItem);
}

// 添加醒目留言（Super Chat）
function addSuperChatMessage(data) {
    const danmuItem = document.createElement('div');
    danmuItem.className = 'danmu-item';
    
    const danmuHeader = document.createElement('div');
    danmuHeader.className = 'danmu-header';
    
    const userName = document.createElement('span');
    userName.className = 'user-name';
    userName.textContent = data.userName || '用户';
    danmuHeader.appendChild(userName);
    
    const price = document.createElement('span');
    price.style.color = '#ff4d4f';
    price.style.marginLeft = '8px';
    price.textContent = `￥${data.price}`;
    danmuHeader.appendChild(price);
    
    const danmuContent = document.createElement('div');
    danmuContent.className = 'danmu-content super-chat';
    danmuContent.textContent = data.content || '';
    
    danmuItem.appendChild(danmuHeader);
    danmuItem.appendChild(danmuContent);
    
    danmuArea.appendChild(danmuItem);
}

// 显示系统消息
function showSystemMessage(message) {
    const danmuItem = document.createElement('div');
    danmuItem.className = 'danmu-item';
    
    const danmuContent = document.createElement('div');
    danmuContent.className = 'danmu-content';
    danmuContent.style.backgroundColor = '#f0f0f0';
    danmuContent.style.color = '#666';
    danmuContent.style.textAlign = 'center';
    danmuContent.textContent = message;
    
    danmuItem.appendChild(danmuContent);
    danmuArea.appendChild(danmuItem);
}

// 更新房间信息
function updateRoomInfo(data) {
    if (data.title) {
        roomTitle.textContent = data.title;
        document.title = `${data.title} - 哔哩哔哩直播`;
    }
    
    if (data.streamerName) {
        streamerName.textContent = `主播: ${data.streamerName}`;
    }
    
    if (data.onlineCount) {
        onlineCount.textContent = formatNumber(data.onlineCount);
    }
    
    if (data.popularity) {
        popularityElement.textContent = formatNumber(data.popularity);
    }
    
    if (data.likes) {
        likesElement.textContent = formatNumber(data.likes);
    }
}

// 更新房间统计数据
function updateRoomStats(data) {
    if (data.onlineCount) {
        onlineCount.textContent = formatNumber(data.onlineCount);
    }
    
    if (data.popularity) {
        popularityElement.textContent = formatNumber(data.popularity);
    }
    
    if (data.likes) {
        likesElement.textContent = formatNumber(data.likes);
    }
}

// 发送弹幕
function sendDanmu() {
    const message = danmuInput.value.trim();
    
    if (!message) {
        return;
    }
    
    fetch('/api/send', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({ message })
    })
    .then(response => response.json())
    .then(data => {
        if (data.status === 'success') {
            danmuInput.value = '';
            // 不需要手动添加弹幕，服务器会通过 WebSocket 发送回来
        } else {
            console.error('发送弹幕失败:', data.error);
            showSystemMessage(`发送失败: ${data.error || '未知错误'}`);
        }
    })
    .catch(error => {
        console.error('发送弹幕请求出错:', error);
        showSystemMessage('发送失败，请重试');
    });
}

// 获取房间信息
function fetchRoomInfo() {
    fetch('/api/room_info')
        .then(response => response.json())
        .then(data => {
            updateRoomInfo(data);
        })
        .catch(error => {
            console.error('获取房间信息失败:', error);
        });
}

// 格式化数字（如: 1000 -> 1,000）
function formatNumber(num) {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', init);