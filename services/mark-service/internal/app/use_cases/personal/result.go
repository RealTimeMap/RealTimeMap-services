package personal

import (
	"time"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/types"
	srv "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/domain/personal"
)

// Имена секций в ответе синхронизации. Совпадают с Name() доменных сервисов.
const (
	SectionMarks  = "personalMarks"
	SectionGroups = "groups"
)

// SyncMarkDTO — персональная метка в ответе синхронизации.
type SyncMarkDTO struct {
	ID       uint `json:"id"`
	Revision uint `json:"revision"`

	Geom types.Point `json:"geom"`

	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Category    string  `json:"category"`

	Color string `json:"color"`
	Icon  string `json:"icon"`

	IsShare   bool `json:"isShare"`
	IsVisible bool `json:"isVisible"`

	GroupIDs []uint   `json:"groupIds"`
	Photos   []string `json:"photos"`
}

// SyncGroupDTO — группа меток в ответе синхронизации.
type SyncGroupDTO struct {
	ID          uint    `json:"id"`
	Revision    uint    `json:"revision"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// ToSyncMarkDTO конвертирует доменную модель в DTO. Вызывается из
// changeSourceAdapter, пока конкретный тип ещё известен статически.
func ToSyncMarkDTO(m srv.Model) any {
	groupIDs := make([]uint, 0, len(m.Groups))
	for _, g := range m.Groups {
		if g == nil {
			continue
		}
		groupIDs = append(groupIDs, g.ID)
	}

	photos := make([]string, 0, len(m.Photos))
	for _, p := range m.Photos {
		photos = append(photos, p.URL)
	}

	return SyncMarkDTO{
		ID:          m.ID,
		Revision:    m.Revision,
		Geom:        m.Geom,
		Title:       m.Title,
		Description: m.Description,
		Category:    m.Category,
		Color:       m.Color,
		Icon:        m.Icon,
		IsShare:     m.IsShare,
		IsVisible:   m.IsVisible,
		GroupIDs:    groupIDs,
		Photos:      photos,
	}
}

func ToSyncGroupDTO(g srv.Group) any {
	return SyncGroupDTO{
		ID:          g.ID,
		Revision:    g.Revision,
		Name:        g.Name,
		Description: g.Description,
	}
}

// PersonalMarkDetailResult — полное представление метки для чтения одной штуки.
type PersonalMarkDetailResult struct {
	ID       uint `json:"id"`
	UserID   uint `json:"userId"`
	Revision uint `json:"revision"`

	Geom types.Point `json:"geom"`

	Title       string  `json:"title"`
	Description *string `json:"description,omitempty"`
	Category    string  `json:"category"`

	Color string `json:"color"`
	Icon  string `json:"icon"`

	IsShare   bool `json:"isShare"`
	IsVisible bool `json:"isVisible"`

	GroupIDs []uint   `json:"groupIds"`
	Photos   []string `json:"photos"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func toPersonalMarkDetailResult(m srv.Model) PersonalMarkDetailResult {
	groupIDs := make([]uint, 0, len(m.Groups))
	for _, g := range m.Groups {
		if g == nil {
			continue
		}
		groupIDs = append(groupIDs, g.ID)
	}

	photos := make([]string, 0, len(m.Photos))
	for _, p := range m.Photos {
		photos = append(photos, p.URL)
	}

	return PersonalMarkDetailResult{
		ID:          m.ID,
		UserID:      m.UserID,
		Revision:    m.Revision,
		Geom:        m.Geom,
		Title:       m.Title,
		Description: m.Description,
		Category:    m.Category,
		Color:       m.Color,
		Icon:        m.Icon,
		IsShare:     m.IsShare,
		IsVisible:   m.IsVisible,
		GroupIDs:    groupIDs,
		Photos:      photos,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}
