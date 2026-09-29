package chatsocket

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zishang520/socket.io/servers/socket/v3"
)

func TestChatIDsFromRooms(t *testing.T) {
	rooms := []socket.Room{
		"Kx8sd0-socket-id",
		UserRoom(7),
		ChatRoom(12),
		ChatRoom(40),
		"chat:",
		"chat:abc",
		"chat:0",
	}

	assert.ElementsMatch(t, []uint{12, 40}, chatIDsFromRooms(rooms))
}
