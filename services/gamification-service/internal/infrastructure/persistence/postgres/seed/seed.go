// Package seed заводит правила начисления опыта и достижения, без которых
// сервис молча ничего не делает: consumer читает события, не находит правила и
// проходит мимо.
//
// Сидер идемпотентен и консервативен. Строка создаётся, только если её ещё
// нет; существующие не трогаются — администратор мог поменять пороги или
// награды через админку, и перезапись затёрла бы эти правки. Как следствие,
// изменение значений в коде не применится к уже засеянной базе: это делается
// миграцией или руками.
package seed

import (
	"context"
	"errors"
	"fmt"

	"github.com/RealTimeMap/RealTimeMap-backend/pkg/transport/kafka/events"
	"github.com/RealTimeMap/RealTimeMap-backend/services/gamification-service/internal/domain/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// rewardSpec — награда, на которую ссылаются правила и достижения.
type rewardSpec struct {
	Code   string
	Amount uint
	Desc   string
}

// achievementSpec — достижение со ступенью цепочки.
//
// Порядок в срезе задаёт цепочку: следующее достижение становится Next для
// предыдущего. Цепочка нужна выдаче ближайших достижений — она показывает
// только текущую ступень, а не все сразу.
type achievementSpec struct {
	Code       string
	Title      string
	Desc       string
	Threshold  uint
	RewardCode string
}

// commentRewards — награды, связанные с комментариями.
var commentRewards = []rewardSpec{
	{Code: "comment_created", Amount: 5, Desc: "Опыт за написанный комментарий"},
	{Code: "ach_first_comment", Amount: 10, Desc: "Награда за первый комментарий"},
	{Code: "ach_commenter_10", Amount: 25, Desc: "Награда за 10 комментариев"},
	{Code: "ach_commenter_50", Amount: 50, Desc: "Награда за 50 комментариев"},
	{Code: "ach_commenter_100", Amount: 100, Desc: "Награда за 100 комментариев"},
}

// commentAchievements — цепочка достижений за комментарии, от ступени к ступени.
var commentAchievements = []achievementSpec{
	{
		Code:       "first_comment",
		Title:      "Первое слово",
		Desc:       "Оставьте свой первый комментарий",
		Threshold:  1,
		RewardCode: "ach_first_comment",
	},
	{
		Code:       "commenter_10",
		Title:      "Собеседник",
		Desc:       "Оставьте 10 комментариев",
		Threshold:  10,
		RewardCode: "ach_commenter_10",
	},
	{
		Code:       "commenter_50",
		Title:      "Активный участник",
		Desc:       "Оставьте 50 комментариев",
		Threshold:  50,
		RewardCode: "ach_commenter_50",
	},
	{
		Code:       "commenter_100",
		Title:      "Голос сообщества",
		Desc:       "Оставьте 100 комментариев",
		Threshold:  100,
		RewardCode: "ach_commenter_100",
	},
}

// commentDailyLimit ограничивает начисление опыта за комментарии в сутки.
//
// Предохранитель от накрутки: без него опыт линейно растёт от числа
// комментариев. На счётчик достижений лимит не влияет — тот считает события, а
// не начисления.
const commentDailyLimit uint = 20

// Run заводит недостающие правила и достижения для событий комментариев.
func Run(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	rewards, err := ensureRewards(ctx, db, logger, commentRewards)
	if err != nil {
		return fmt.Errorf("seed rewards: %w", err)
	}

	if err := ensureEventRule(ctx, db, logger, model.EventRule{
		EventType:      events.CommentCreated,
		KafkaEventType: events.CommentCreated,
		Description:    strPtr("Опыт за написание комментария"),
		RewardID:       rewards["comment_created"],
		IsActive:       true,
		IsRepeatable:   true,
		DailyLimit:     uintPtr(commentDailyLimit),
	}); err != nil {
		return fmt.Errorf("seed event rule: %w", err)
	}

	if err := ensureAchievementChain(ctx, db, logger, commentAchievements, rewards); err != nil {
		return fmt.Errorf("seed achievements: %w", err)
	}

	return nil
}

// ensureRewards создаёт недостающие награды и возвращает их id по коду.
func ensureRewards(ctx context.Context, db *gorm.DB, logger *zap.Logger, specs []rewardSpec) (map[string]uint, error) {
	ids := make(map[string]uint, len(specs))

	for _, spec := range specs {
		var reward model.XPReward
		err := db.WithContext(ctx).Where("code = ?", spec.Code).First(&reward).Error

		switch {
		case err == nil:
			ids[spec.Code] = reward.ID
		case errors.Is(err, gorm.ErrRecordNotFound):
			reward = model.XPReward{
				Code:        spec.Code,
				Amount:      spec.Amount,
				Description: strPtr(spec.Desc),
			}
			if err := db.WithContext(ctx).Create(&reward).Error; err != nil {
				return nil, err
			}
			logger.Info("seeded xp reward", zap.String("code", spec.Code), zap.Uint("amount", spec.Amount))
			ids[spec.Code] = reward.ID
		default:
			return nil, err
		}
	}

	return ids, nil
}

func ensureEventRule(ctx context.Context, db *gorm.DB, logger *zap.Logger, rule model.EventRule) error {
	var existing model.EventRule
	err := db.WithContext(ctx).Where("event_type = ?", rule.EventType).First(&existing).Error

	switch {
	case err == nil:
		return nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := db.WithContext(ctx).Create(&rule).Error; err != nil {
			return err
		}
		logger.Info("seeded event rule", zap.String("event_type", rule.EventType))
		return nil
	default:
		return err
	}
}

// ensureAchievementChain создаёт достижения и связывает их в цепочку.
//
// Связывание идёт вторым проходом: Next ссылается на достижение, которого при
// создании предыдущего ещё нет.
func ensureAchievementChain(
	ctx context.Context,
	db *gorm.DB,
	logger *zap.Logger,
	specs []achievementSpec,
	rewards map[string]uint,
) error {
	ids := make([]uint, len(specs))

	for i, spec := range specs {
		rewardID, ok := rewards[spec.RewardCode]
		if !ok {
			return fmt.Errorf("reward %q not found for achievement %q", spec.RewardCode, spec.Code)
		}

		var existing model.Achievement
		err := db.WithContext(ctx).Where("code = ?", spec.Code).First(&existing).Error

		switch {
		case err == nil:
			ids[i] = existing.ID
		case errors.Is(err, gorm.ErrRecordNotFound):
			created := model.Achievement{
				Code:             spec.Code,
				Title:            spec.Title,
				Desc:             spec.Desc,
				TriggerEventType: events.CommentCreated,
				Threshold:        spec.Threshold,
				RewardID:         rewardID,
				IsActive:         true,
			}
			if err := db.WithContext(ctx).Create(&created).Error; err != nil {
				return err
			}
			logger.Info("seeded achievement",
				zap.String("code", spec.Code),
				zap.Uint("threshold", spec.Threshold),
			)
			ids[i] = created.ID
		default:
			return err
		}
	}

	// Второй проход: связываем ступени. Обновляем только пустой next_id, чтобы
	// не перетереть ручную перестройку цепочки.
	for i := 0; i < len(ids)-1; i++ {
		next := ids[i+1]
		if err := db.WithContext(ctx).
			Model(&model.Achievement{}).
			Where("id = ? AND next_id IS NULL", ids[i]).
			Update("next_id", next).Error; err != nil {
			return err
		}
	}

	return nil
}

func strPtr(s string) *string { return &s }

func uintPtr(v uint) *uint { return &v }
