package template

import "github.com/RealTimeMap/RealTimeMap-backend/pkg/apperror"

// ErrNotFound — запрошенного шаблона не существует.
//
// Терминальная ошибка: в MVP шаблоны лежат в embed.FS и сами не появятся,
// повторная попытка отправки бессмысленна.
func ErrNotFound(templateID string) error {
	return apperror.NewNotFoundErrorByID("template", templateID)
}

// ErrMissingData — в данных нет поля, объявленного шаблоном обязательным.
func ErrMissingData(field string) error {
	return apperror.NewRequiredError(field)
}
