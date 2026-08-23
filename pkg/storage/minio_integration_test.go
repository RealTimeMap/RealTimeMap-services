package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/logger"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Интеграционные тесты хранилища: проверяют ровно то, что нельзя проверить без
// живого MinIO — дедупликацию через StatObject, ранний выход без записи,
// поведение PutObject и отличие «объекта нет» от ошибки доступа.
//
// Требуют поднятого MinIO. Без него пропускаются. Запуск из корня репозитория:
//
//	go test ./pkg/storage/ -run Integration -v
//
// Параметры подключения берутся из окружения, значения по умолчанию совпадают
// с docker-compose (проброс 9000 на хост):
//
//	STORAGE_TEST_ENDPOINT   (по умолчанию localhost:9000)
//	STORAGE_TEST_ACCESS_KEY (по умолчанию из MINIO_ROOT_USER)
//	STORAGE_TEST_SECRET_KEY (по умолчанию из MINIO_ROOT_PASSWORD)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func testCreds(t *testing.T) (endpoint, access, secret string) {
	t.Helper()
	endpoint = envOr("STORAGE_TEST_ENDPOINT", "localhost:9000")
	access = envOr("STORAGE_TEST_ACCESS_KEY", os.Getenv("MINIO_ROOT_USER"))
	secret = envOr("STORAGE_TEST_SECRET_KEY", os.Getenv("MINIO_ROOT_PASSWORD"))
	if access == "" || secret == "" {
		t.Skip("нет ключей MinIO: задай STORAGE_TEST_ACCESS_KEY/STORAGE_TEST_SECRET_KEY или MINIO_ROOT_USER/MINIO_ROOT_PASSWORD")
	}
	return endpoint, access, secret
}

// setupStorage поднимает Storage поверх ОТДЕЛЬНОГО временного бакета: тесты не
// должны мусорить в рабочем, а ключи content-addressed, поэтому удалять «свои»
// объекты из общего бакета нельзя — они могут разделяться с чужими данными.
func setupStorage(t *testing.T) (Storage, *minio.Client, string) {
	t.Helper()

	endpoint, access, secret := testCreds(t)

	ctx := context.Background()
	client, err := minio.New(endpoint, &minio.Options{
		Creds: credentials.NewStaticV4(access, secret, ""),
	})
	if err != nil {
		t.Skipf("MinIO недоступен: %v", err)
	}

	bucket := "test-storage-" + strings.Split(uuid.New().String(), "-")[0]
	if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		t.Skipf("MinIO недоступен (%s): %v", endpoint, err)
	}

	t.Cleanup(func() {
		for obj := range client.ListObjects(ctx, bucket, minio.ListObjectsOptions{Recursive: true}) {
			if obj.Err == nil {
				_ = client.RemoveObject(ctx, bucket, obj.Key, minio.RemoveObjectOptions{})
			}
		}
		_ = client.RemoveBucket(ctx, bucket)
	})

	s, err := NewMinIOStorage(StorageConfig{
		Endpoint:        endpoint,
		Bucket:          bucket,
		AccessKeyID:     access,
		SecretAccessKey: secret,
		Region:          "us-east-1",
		UseSSL:          false,
		BaseURL:         "https://example.test/files",
	}, logger.NewNop())
	require.NoError(t, err)

	return s, client, bucket
}

// makeJPEG собирает валидный JPEG заданного размера. Байты должны быть
// настоящими: Upload определяет mime по магическим байтам и читает заголовок
// ради размеров.
func makeJPEG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}))
	return buf.Bytes()
}

func makePNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func defaultOpts() UploadOptions {
	return UploadOptions{
		FileName: "test.jpg",
		Category: CategoryMarkPhoto,
		MaxSize:  5 << 20,
	}
}

func TestIntegrationUpload(t *testing.T) {
	s, client, bucket := setupStorage(t)
	ctx := context.Background()

	t.Run("заливка возвращает заполненную Photo", func(t *testing.T) {
		data := makeJPEG(t, 120, 80, color.RGBA{R: 200, A: 255})

		photo, err := s.Upload(ctx, data, defaultOpts())
		require.NoError(t, err)

		sum := sha256.Sum256(data)
		wantHash := hex.EncodeToString(sum[:])

		assert.Equal(t, wantHash, photo.Hash, "Hash — SHA256 исходных байтов")
		assert.Equal(t, buildKey(wantHash, ".jpg"), photo.StorageKey)
		assert.Equal(t, int64(len(data)), photo.Size)
		assert.Equal(t, 120, photo.Width)
		assert.Equal(t, 80, photo.Height)
		assert.Equal(t, "image/jpeg", photo.MimeType)
		assert.Equal(t, "test.jpg", photo.FileName)
		assert.Equal(t, "https://example.test/files/"+photo.StorageKey, photo.URL)
		assert.False(t, photo.UploadedAt.IsZero())

		// Объект действительно лежит в бакете.
		stat, err := client.StatObject(ctx, bucket, photo.StorageKey, minio.StatObjectOptions{})
		require.NoError(t, err)
		assert.Equal(t, int64(len(data)), stat.Size)
	})

	t.Run("категория и имя файла уходят в user-metadata", func(t *testing.T) {
		data := makeJPEG(t, 15, 15, color.RGBA{G: 120, A: 255})
		opts := defaultOpts()
		opts.Category = CategoryProfileAvatar
		opts.FileName = "avatar.jpg"
		opts.Metadata = map[string]string{"custom": "value"}

		photo, err := s.Upload(ctx, data, opts)
		require.NoError(t, err)

		stat, err := client.StatObject(ctx, bucket, photo.StorageKey, minio.StatObjectOptions{})
		require.NoError(t, err)
		assert.Equal(t, "avatars", stat.UserMetadata["Category"])
		assert.Equal(t, "avatar.jpg", stat.UserMetadata["Filename"])
		assert.Equal(t, "value", stat.UserMetadata["Custom"])
		assert.Equal(t, "image/jpeg", stat.ContentType)
	})

	t.Run("расширение берётся из mime, если имя файла без него", func(t *testing.T) {
		data := makePNG(t, 10, 10)
		opts := defaultOpts()
		opts.FileName = "no-extension"

		photo, err := s.Upload(ctx, data, opts)
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(photo.StorageKey, ".png"), photo.StorageKey)
		assert.Equal(t, "image/png", photo.MimeType)
	})
}

func TestIntegrationUploadDeduplication(t *testing.T) {
	s, client, bucket := setupStorage(t)
	ctx := context.Background()
	data := makeJPEG(t, 100, 60, color.RGBA{B: 200, A: 255})

	first, err := s.Upload(ctx, data, defaultOpts())
	require.NoError(t, err)

	statBefore, err := client.StatObject(ctx, bucket, first.StorageKey, minio.StatObjectOptions{})
	require.NoError(t, err)

	t.Run("повторная заливка не пишет в бакет", func(t *testing.T) {
		second, err := s.Upload(ctx, data, defaultOpts())
		require.NoError(t, err)

		assert.Equal(t, first.StorageKey, second.StorageKey)
		assert.Equal(t, first.Hash, second.Hash)

		statAfter, err := client.StatObject(ctx, bucket, second.StorageKey, minio.StatObjectOptions{})
		require.NoError(t, err)
		// Главная проверка дедупликации: объект не перезаписан.
		assert.Equal(t, statBefore.LastModified, statAfter.LastModified,
			"LastModified изменился — значит был лишний PutObject")
		assert.Equal(t, statBefore.ETag, statAfter.ETag)
	})

	t.Run("на пути раннего выхода размеры не теряются", func(t *testing.T) {
		second, err := s.Upload(ctx, data, defaultOpts())
		require.NoError(t, err)
		// Ранний выход не читает объект из бакета: размеры считаются из
		// переданных байтов и обязаны совпадать с первой заливкой.
		assert.Equal(t, first.Width, second.Width)
		assert.Equal(t, first.Height, second.Height)
		assert.NotZero(t, second.Width)
	})

	t.Run("Size на раннем выходе берётся из StatObject", func(t *testing.T) {
		second, err := s.Upload(ctx, data, defaultOpts())
		require.NoError(t, err)
		assert.Equal(t, statBefore.Size, second.Size)
	})

	t.Run("дедуп не зависит от категории", func(t *testing.T) {
		opts := defaultOpts()
		opts.Category = CategoryCommentPhoto

		other, err := s.Upload(ctx, data, opts)
		require.NoError(t, err)
		assert.Equal(t, first.StorageKey, other.StorageKey,
			"категория не входит в ключ, дедуп обязан работать между категориями")
	})

	t.Run("разное содержимое даёт разные ключи", func(t *testing.T) {
		other := makeJPEG(t, 100, 60, color.RGBA{R: 1, G: 2, B: 3, A: 255})
		photo, err := s.Upload(ctx, other, defaultOpts())
		require.NoError(t, err)
		assert.NotEqual(t, first.StorageKey, photo.StorageKey)
	})
}

func TestIntegrationUploadOptimize(t *testing.T) {
	s, _, _ := setupStorage(t)
	ctx := context.Background()
	data := makeJPEG(t, 90, 70, color.RGBA{R: 10, G: 180, B: 40, A: 255})

	optimized := defaultOpts()
	optimized.Optimize = true

	withOpt, err := s.Upload(ctx, data, optimized)
	require.NoError(t, err)

	t.Run("ключ считается от исходных байтов, не от результата", func(t *testing.T) {
		// Ключевое свойство: иначе ранний выход не срабатывал бы при
		// Optimize: true и файл перекодировался бы на каждой заливке.
		sum := sha256.Sum256(data)
		assert.Equal(t, hex.EncodeToString(sum[:]), withOpt.Hash)

		plain, err := s.Upload(ctx, data, defaultOpts())
		require.NoError(t, err)
		assert.Equal(t, withOpt.StorageKey, plain.StorageKey,
			"Optimize не должен менять ключ — иначе дедуп сломан")
	})

	t.Run("размеры берутся из исходника и не искажаются", func(t *testing.T) {
		assert.Equal(t, 90, withOpt.Width)
		assert.Equal(t, 70, withOpt.Height)
	})
}

func TestIntegrationUploadValidation(t *testing.T) {
	s, _, _ := setupStorage(t)
	ctx := context.Background()

	t.Run("невалидная категория", func(t *testing.T) {
		opts := defaultOpts()
		opts.Category = "unknown"
		_, err := s.Upload(ctx, makeJPEG(t, 10, 10, color.White), opts)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidCategory))
	})

	t.Run("превышение MaxSize", func(t *testing.T) {
		opts := defaultOpts()
		opts.MaxSize = 50
		_, err := s.Upload(ctx, makeJPEG(t, 200, 200, color.White), opts)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrFileTooLarge))
	})

	t.Run("MaxSize=0 означает без лимита", func(t *testing.T) {
		opts := defaultOpts()
		opts.MaxSize = 0
		_, err := s.Upload(ctx, makeJPEG(t, 40, 40, color.RGBA{R: 77, A: 255}), opts)
		assert.NoError(t, err)
	})

	t.Run("не-картинка отклоняется по магическим байтам", func(t *testing.T) {
		_, err := s.Upload(ctx, []byte("plain text, not an image"), defaultOpts())
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrInvalidMimeType))
	})

	t.Run("явный MimeType принимается на веру", func(t *testing.T) {
		opts := defaultOpts()
		// Содержимое не является картинкой, но MimeType задан явно и по
		// содержимому не перепроверяется. Фиксируем текущее поведение, чтобы
		// его изменение было осознанным: валидация входа живёт в
		// pkg/mediavalidator, а вызывающий код зовёт её до Upload.
		opts.MimeType = "image/jpeg"
		photo, err := s.Upload(ctx, []byte("not an image"), opts)
		require.NoError(t, err)
		assert.Zero(t, photo.Width, "размеры не читаются: заголовка изображения нет")
	})
}

func TestIntegrationUploadMultiple(t *testing.T) {
	s, _, _ := setupStorage(t)
	ctx := context.Background()

	t.Run("порядок результата совпадает с порядком входа", func(t *testing.T) {
		files := []FileUpload{
			{Data: makeJPEG(t, 11, 11, color.RGBA{R: 1, A: 255}), Options: defaultOpts()},
			{Data: makeJPEG(t, 22, 22, color.RGBA{R: 2, A: 255}), Options: defaultOpts()},
			{Data: makeJPEG(t, 33, 33, color.RGBA{R: 3, A: 255}), Options: defaultOpts()},
			{Data: makeJPEG(t, 44, 44, color.RGBA{R: 4, A: 255}), Options: defaultOpts()},
		}

		photos, err := s.UploadMultiple(ctx, files)
		require.NoError(t, err)
		require.Len(t, photos, len(files))

		for i, want := range []int{11, 22, 33, 44} {
			assert.Equal(t, want, photos[i].Width, "позиция %d", i)
		}
	})

	t.Run("fail-fast: ошибка одного файла рушит весь вызов", func(t *testing.T) {
		bad := defaultOpts()
		bad.MaxSize = 10

		files := []FileUpload{
			{Data: makeJPEG(t, 12, 12, color.RGBA{B: 9, A: 255}), Options: defaultOpts()},
			{Data: makeJPEG(t, 300, 300, color.RGBA{B: 8, A: 255}), Options: bad},
		}

		photos, err := s.UploadMultiple(ctx, files)
		require.Error(t, err, "раньше ошибка проглатывалась и возвращался частичный результат")
		assert.Nil(t, photos, "частичный результат отдавать нельзя")
		assert.True(t, errors.Is(err, ErrFileTooLarge))
		assert.Contains(t, err.Error(), "file 1", "в ошибке должен быть индекс файла")
	})

	t.Run("пустой список — пустой результат без ошибки", func(t *testing.T) {
		photos, err := s.UploadMultiple(ctx, nil)
		require.NoError(t, err)
		assert.Empty(t, photos)
	})

	t.Run("дубликаты внутри одного вызова дают один ключ", func(t *testing.T) {
		data := makeJPEG(t, 25, 25, color.RGBA{G: 200, A: 255})
		files := []FileUpload{
			{Data: data, Options: defaultOpts()},
			{Data: data, Options: defaultOpts()},
		}

		photos, err := s.UploadMultiple(ctx, files)
		require.NoError(t, err)
		require.Len(t, photos, 2)
		assert.Equal(t, photos[0].StorageKey, photos[1].StorageKey)
	})

	t.Run("файлов больше, чем воркеров", func(t *testing.T) {
		files := make([]FileUpload, maxWorkers*2+1)
		for i := range files {
			files[i] = FileUpload{
				Data:    makeJPEG(t, 10+i, 10+i, color.RGBA{R: uint8(i), A: 255}),
				Options: defaultOpts(),
			}
		}

		photos, err := s.UploadMultiple(ctx, files)
		require.NoError(t, err)
		require.Len(t, photos, len(files))
		for i := range files {
			assert.Equal(t, 10+i, photos[i].Width, "позиция %d", i)
		}
	})
}

func TestIntegrationExists(t *testing.T) {
	s, _, _ := setupStorage(t)
	ctx := context.Background()

	photo, err := s.Upload(ctx, makeJPEG(t, 30, 30, color.RGBA{R: 5, G: 5, A: 255}), defaultOpts())
	require.NoError(t, err)

	t.Run("существующий объект", func(t *testing.T) {
		ok, err := s.Exists(ctx, photo.StorageKey)
		require.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("несуществующий объект: false без ошибки", func(t *testing.T) {
		// Отличие «нет объекта» от настоящей ошибки принципиально: спутав их,
		// Upload посчитал бы существующий файл отсутствующим.
		ok, err := s.Exists(ctx, "v1/00/00/"+strings.Repeat("0", 64)+".jpg")
		require.NoError(t, err)
		assert.False(t, ok)
	})
}

func TestIntegrationNewMinIOStorage(t *testing.T) {
	endpoint, access, secret := testCreds(t)

	t.Run("несуществующий бакет — ошибка на старте, а не при первой заливке", func(t *testing.T) {
		_, err := NewMinIOStorage(StorageConfig{
			Endpoint:        endpoint,
			Bucket:          "missing-bucket-" + strings.Split(uuid.New().String(), "-")[0],
			AccessKeyID:     access,
			SecretAccessKey: secret,
			UseSSL:          false,
		}, logger.NewNop())
		require.Error(t, err)
	})

	t.Run("неверные ключи — ошибка", func(t *testing.T) {
		_, err := NewMinIOStorage(StorageConfig{
			Endpoint:        endpoint,
			Bucket:          "rtm-media",
			AccessKeyID:     "wrong-access-key",
			SecretAccessKey: "wrong-secret-key",
			UseSSL:          false,
		}, logger.NewNop())
		require.Error(t, err)
	})
}
