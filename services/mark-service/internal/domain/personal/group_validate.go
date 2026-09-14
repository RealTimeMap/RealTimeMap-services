package personal

import "regexp"

// maxGroupIconLen — предел длины имени иконки iconfy ("mdi:coffee-outline").
// Совпадает с size:128 у колонки icon.
const maxGroupIconLen = 128

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// validateGroupColor проверяет hex-код цвета группы. Пустая строка допустима:
// цвет необязателен, клиент подставит собственный дефолт.
func validateGroupColor(color string) error {
	if color == "" {
		return nil
	}
	if !hexColorRegex.MatchString(color) {
		return ErrGroupColorInvalid(color)
	}
	return nil
}

// validateGroupIcon проверяет имя иконки iconfy. Пустая строка допустима.
func validateGroupIcon(icon string) error {
	if len(icon) > maxGroupIconLen {
		return ErrGroupIconInvalid(icon)
	}
	return nil
}
