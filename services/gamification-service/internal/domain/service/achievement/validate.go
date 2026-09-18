package achievement

import (
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/domainerrors"
)

// maxIconLen — предел длины имени иконки iconfy ("mdi:trophy-outline").
// Совпадает с size:128 у колонки icon.
const maxIconLen = 128

// validateIcon проверяет имя иконки iconfy.
//
// Иконка обязательна: достижение без неё нечем отрисовать в списке, а раньше
// эту роль играл обязательный файл.
func validateIcon(icon string) error {
	if icon == "" {
		return domainerrors.ErrAchievementIconRequired()
	}
	if len(icon) > maxIconLen {
		return domainerrors.ErrAchievementIconInvalid(icon)
	}
	return nil
}
