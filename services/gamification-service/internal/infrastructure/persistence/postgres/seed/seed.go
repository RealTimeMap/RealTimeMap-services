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

// bugRewards — награды за баги, подтверждённые разработчиком.
//
// Опыт за подтверждённый баг выше, чем за комментарий: засчитывается только
// отчёт, который разработчик воспроизвёл, а большинство отчётов проверку не
// проходит.
var bugRewards = []rewardSpec{
	{Code: "bug_confirmed", Amount: 30, Desc: "Опыт за подтверждённый баг"},
	{Code: "ach_first_bug", Amount: 20, Desc: "Награда за первый подтверждённый баг"},
	{Code: "ach_bug_hunter_5", Amount: 50, Desc: "Награда за 5 подтверждённых багов"},
	{Code: "ach_bug_hunter_15", Amount: 100, Desc: "Награда за 15 подтверждённых багов"},
	{Code: "ach_bug_hunter_30", Amount: 200, Desc: "Награда за 30 подтверждённых багов"},
}

// bugAchievements — цепочка достижений за подтверждённые баги.
var bugAchievements = []achievementSpec{
	{
		Code:       "first_bug",
		Title:      "Внимательный глаз",
		Desc:       "Сообщите о баге, который подтвердит разработчик",
		Threshold:  1,
		RewardCode: "ach_first_bug",
	},
	{
		Code:       "bug_hunter_5",
		Title:      "Охотник за багами",
		Desc:       "Найдите 5 подтверждённых багов",
		Threshold:  5,
		RewardCode: "ach_bug_hunter_5",
	},
	{
		Code:       "bug_hunter_15",
		Title:      "Тестировщик",
		Desc:       "Найдите 15 подтверждённых багов",
		Threshold:  15,
		RewardCode: "ach_bug_hunter_15",
	},
	{
		Code:       "bug_hunter_30",
		Title:      "Страж качества",
		Desc:       "Найдите 30 подтверждённых багов",
		Threshold:  30,
		RewardCode: "ach_bug_hunter_30",
	},
}

// commentDailyLimit ограничивает начисление опыта за комментарии в сутки.
//
// Предохранитель от накрутки: без него опыт линейно растёт от числа
// комментариев. На счётчик достижений лимит не влияет — тот считает события, а
// не начисления.
const commentDailyLimit uint = 20

// Run заводит недостающие правила и достижения для событий комментариев и
// подтверждённых багов.
func Run(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	if err := seedComments(ctx, db, logger); err != nil {
		return err
	}
	if err := seedBugs(ctx, db, logger); err != nil {
		return err
	}
	return nil
}

func seedComments(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
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

	if err := ensureAchievementChain(ctx, db, logger, events.CommentCreated, commentAchievements, rewards); err != nil {
		return fmt.Errorf("seed achievements: %w", err)
	}

	return nil
}

// seedBugs заводит правило и достижения за подтверждённые баги.
//
// Правило обязательно, хотя цель — достижения: счётчик, по которому они
// открываются, растёт только вместе с начислением опыта по правилу.
//
// Дневного лимита нет: событие шлёт не пользователь, а разработчик,
// подтвердив отчёт, — накрутить его автор не может. Повторы отсекает сам
// feedback-service: событие уходит только при первом подтверждении бага.
func seedBugs(ctx context.Context, db *gorm.DB, logger *zap.Logger) error {
	rewards, err := ensureRewards(ctx, db, logger, bugRewards)
	if err != nil {
		return fmt.Errorf("seed bug rewards: %w", err)
	}

	if err := ensureEventRule(ctx, db, logger, model.EventRule{
		EventType:      events.BugConfirmed,
		KafkaEventType: events.BugConfirmed,
		Description:    strPtr("Опыт за баг, подтверждённый разработчиком"),
		RewardID:       rewards["bug_confirmed"],
		IsActive:       true,
		IsRepeatable:   true,
	}); err != nil {
		return fmt.Errorf("seed bug event rule: %w", err)
	}

	if err := ensureAchievementChain(ctx, db, logger, events.BugConfirmed, bugAchievements, rewards); err != nil {
		return fmt.Errorf("seed bug achievements: %w", err)
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
	triggerEvent string,
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
				TriggerEventType: triggerEvent,
				Threshold:        spec.Threshold,
				RewardID:         rewardID,
				IsActive:         true,
			}
			if err := db.WithContext(ctx).Create(&created).Error; err != nil {
				return err
			}
			logger.Info("seeded achievement",
				zap.String("code", spec.Code),
				zap.String("trigger_event", triggerEvent),
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
