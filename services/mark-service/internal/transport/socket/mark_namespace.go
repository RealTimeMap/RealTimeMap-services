package socket

import (
	"context"
	"encoding/json"
	"time"

	subdto "github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/dto/mark"
	"github.com/RealTimeMap/RealTimeMap-backend/services/mark-service/internal/transport/http/dto/mark"
	"github.com/doquangtan/socketio/v4"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// InitMarkNamespace иницилизирует mark Namespace
// Позволяет работать с метками в релаьном времени
// Ивенты Client -> Server
// message - дефолтный ивент для обработки новых параметров фильтрации
// Ивенты Server -> Client
// markCreated - создание новой метки
func InitMarkNamespace(s *SocketServer) {
	ns := s.io.Of("/marks")
	s.logger.Info("init mark namespace", zap.String("namespace", ns.Name))

	ns.OnConnection(func(socket *socketio.Socket) {
		socket.On("message", func(event *socketio.EventPayload) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if len(event.Data) == 0 {
				return
			}
			rawData := event.Data[0]

			params, err := parseAndValidate(rawData)
			if err != nil {
				s.logger.Warn("failed to validate params", zap.Error(err))
				if event.Ack != nil {
					event.Ack(map[string]interface{}{
						"success": false,
						"error":   err.Error(),
					})
				}
				return
			}
			validParams := subdto.ToInputFilter(params)

			if validParams.ZoomLevel < 12 {
				clusters, err := s.markService.GetMarksInCluster(ctx, validParams)
				if err != nil {
					s.logger.Warn("failed to get cluster", zap.Error(err))
					if event.Ack != nil {
						event.Ack(map[string]interface{}{
							"success": false,
							"error":   err.Error(),
						})
					}
					return
				}
				clusterResponse := mark.NewMultipleResponseCluster(clusters)
				event.Ack(map[string]interface{}{
					"success": true,
					"cluster": clusterResponse,
				})
				return
			} else {
				marks, err := s.markService.GetMarksInArea(ctx, validParams)
				if err != nil {
					s.logger.Warn("failed to get cluster", zap.Error(err))
					if event.Ack != nil {
						event.Ack(map[string]interface{}{
							"success": false,
							"error":   err,
						})
					}
					return
				}
				marksResponse := mark.NewMultipleResponseMark(marks)
				event.Ack(map[string]interface{}{
					"success": true,
					"marks":   marksResponse,
				})
				return
			}

			// TODO принимаем данные для фильтрации
		})
		socket.On("markCreated", func(data *socketio.EventPayload) { // TODO После реализации функционала поменять на Server -> Client
			// TODO Уведомляем пользователя о новой метке в его зоне
			// TODO сделать фильтрацию с bbox
		})
		socket.On("markDeleted", func(data *socketio.EventPayload) { // TODO После реализации функционала поменять на Server -> Client
			// TODO Уведомляем пользователя о удалении метке в его зоне
			// TODO сделать фильтрацию с bbox
		})
		socket.On("markUpdated", func(data *socketio.EventPayload) { // TODO После реализации функционала поменять на Server -> Client
			// TODO Уведомляем пользователя о обновлении меткм в его зоне
			// TODO сделать фильтрацию с bbox
		})
	})

}

func parseAndValidate(data interface{}) (subdto.FilterParams, error) {
	var params subdto.FilterParams
	params.ZoomLevel = 12
	params.EndAt = time.Now().UTC()

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return subdto.FilterParams{}, err
	}

	// Сериализуем в структуру
	if err := json.Unmarshal(jsonBytes, &params); err != nil {
		return subdto.FilterParams{}, err
	}
	if err := validate.Struct(params); err != nil {
		return subdto.FilterParams{}, err
	}
	return params, nil
}
