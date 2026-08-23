package storage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Юнит-тесты чистых функций пакета: ключ, валидация категорий и mime.
// MinIO здесь не нужен — всё, что требует живого хранилища, лежит в
// minio_integration_test.go.

func TestBuildKey(t *testing.T) {
	const hash = "abcd1234ef567890abcd1234ef567890abcd1234ef567890abcd1234ef567890"

	t.Run("шардирование по первым 4 символам хеша", func(t *testing.T) {
		assert.Equal(t, "v1/ab/cd/"+hash+".jpg", buildKey(hash, ".jpg"))
	})

	t.Run("расширение подставляется как есть", func(t *testing.T) {
		assert.Equal(t, "v1/ab/cd/"+hash+".png", buildKey(hash, ".png"))
		assert.Equal(t, "v1/ab/cd/"+hash+".webp", buildKey(hash, ".webp"))
	})

	t.Run("пустое расширение не ломает ключ", func(t *testing.T) {
		assert.Equal(t, "v1/ab/cd/"+hash, buildKey(hash, ""))
	})

	t.Run("префикс v1 обязателен: по нему открыт публичный доступ", func(t *testing.T) {
		// Бакет публичен на чтение ТОЛЬКО для v1/ (см. minio-init в
		// docker-compose). Ключ без этого префикса вернёт 403 клиенту.
		assert.True(t, len(buildKey(hash, ".jpg")) > 3 && buildKey(hash, ".jpg")[:3] == "v1/")
	})

	t.Run("разные хеши дают разные шарды", func(t *testing.T) {
		other := "ffee1234ef567890abcd1234ef567890abcd1234ef567890abcd1234ef567890"
		assert.NotEqual(t, buildKey(hash, ".jpg"), buildKey(other, ".jpg"))
		assert.Contains(t, buildKey(other, ".jpg"), "v1/ff/ee/")
	})
}

func TestCategoryStorage_Validate(t *testing.T) {
	valid := []CategoryStorage{
		CategoryMarkPhoto,
		CategoryCommentPhoto,
		CategoryTemp,
		CategoryCategories,
		CategoryProfileAvatar,
		CategoryAchievement,
	}
	for _, c := range valid {
		t.Run("валидная: "+c.String(), func(t *testing.T) {
			assert.NoError(t, c.Validate())
		})
	}

	invalid := []CategoryStorage{"", "unknown", "Marks", "marks "}
	for _, c := range invalid {
		t.Run("невалидная: "+string(c), func(t *testing.T) {
			require.Error(t, c.Validate())
			assert.True(t, errors.Is(c.Validate(), ErrInvalidCategory))
		})
	}
}

func TestCategoryStorage_String(t *testing.T) {
	assert.Equal(t, "marks", CategoryMarkPhoto.String())
	assert.Equal(t, "avatars", CategoryProfileAvatar.String())
	assert.Equal(t, "achievements", CategoryAchievement.String())
}

func TestIsValidMimeType(t *testing.T) {
	t.Run("разрешённые", func(t *testing.T) {
		for _, m := range []string{"image/jpeg", "image/png", "image/gif", "image/webp", "video/mp4", "video/webm"} {
			assert.True(t, isValidMimeType(m), m)
		}
	})

	t.Run("запрещённые", func(t *testing.T) {
		for _, m := range []string{"", "application/octet-stream", "text/plain", "application/pdf", "image/svg+xml", "IMAGE/JPEG"} {
			assert.False(t, isValidMimeType(m), m)
		}
	})

	t.Run("image/svg+xml отклоняется: SVG может содержать скрипт", func(t *testing.T) {
		assert.False(t, isValidMimeType("image/svg+xml"))
	})
}

func TestIsImage(t *testing.T) {
	for _, m := range []string{"image/jpeg", "image/png", "image/gif", "image/webp"} {
		assert.True(t, isImage(m), m)
	}
	// Видео проходит isValidMimeType, но НЕ является изображением: Optimize
	// к нему применять нельзя.
	for _, m := range []string{"video/mp4", "video/webm", "application/octet-stream", ""} {
		assert.False(t, isImage(m), m)
	}
}

func TestGetURL(t *testing.T) {
	s := &MinIOStorage{baseURL: "https://realtimemap.ru/files"}
	assert.Equal(t,
		"https://realtimemap.ru/files/v1/ab/cd/hash.jpg",
		s.GetURL("v1/ab/cd/hash.jpg"),
	)
}
