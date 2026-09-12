package template

import (
	"context"
	"maps"
	"strings"
	"testing"

	domain "github.com/RealTimeMap/RealTimeMap-backend/services/smtp-service/internal/domain/template"
	"github.com/stretchr/testify/require"
)

// nextAchievement повторяет форму, которую шаблон ожидает в nextAchievements.
type nextAchievement struct {
	Title            string
	Condition        string
	Current          uint
	Threshold        uint
	Percent          uint
	RemainingPercent uint
}

func newRenderer(t *testing.T) *domain.Renderer {
	t.Helper()
	p, err := NewProvider()
	require.NoError(t, err)
	return domain.NewRenderer(p)
}

func TestCommentReplyRenders(t *testing.T) {
	out, err := newRenderer(t).Render(context.Background(), "commentReply", nil, map[string]any{
		"username":                "winerty",
		"authorName":              "Аноним <script>alert(1)</script>",
		"commentText":             "Отличное место, был там вчера!",
		"commentUrl":              "https://realtimemap.ru/marks/42#comment-7",
		"notificationSettingsUrl": "https://realtimemap.ru/settings/notifications",
	})
	require.NoError(t, err)

	require.Equal(t, "Аноним &lt;script&gt;alert(1)&lt;/script&gt; ответил на ваш комментарий", out.Subject)
	require.Contains(t, out.HTML, "Отличное место")
	// html/template экранирует данные по контексту: имя вида "<script>"
	// попадает в письмо текстом, а не разметкой.
	require.NotContains(t, out.HTML, "<script>alert(1)</script>")
}

func TestAchievementRendersWithNextSteps(t *testing.T) {
	out, err := newRenderer(t).Render(context.Background(), "achievementUnlocked", nil, achievementData(map[string]any{
		"nextAchievements": []nextAchievement{
			{Title: "Собеседник", Condition: "10 комментариев", Current: 1, Threshold: 10, Percent: 10, RemainingPercent: 90},
			{Title: "Активный участник", Condition: "50 комментариев", Current: 1, Threshold: 50, Percent: 2, RemainingPercent: 98},
		},
	}))
	require.NoError(t, err)

	require.Equal(t, "Новое достижение: Первое слово", out.Subject)
	require.Contains(t, out.HTML, "Первое слово")
	require.Contains(t, out.HTML, "СЛЕДУЮЩЕЕ ПО ПУТИ")
	require.Contains(t, out.HTML, "Собеседник")
	require.Contains(t, out.HTML, "Активный участник")
	require.Contains(t, out.HTML, "1 / 10")
	// Ширина заполненной части приходит готовым числом — арифметики в
	// html/template нет.
	require.Contains(t, out.HTML, "width:10%")
}

// На последней ступени цепочки показывать нечего — секция должна исчезнуть,
// а не отрендериться пустой рамкой.
func TestAchievementHidesNextStepsWhenEmpty(t *testing.T) {
	out, err := newRenderer(t).Render(context.Background(), "achievementUnlocked", nil, achievementData(nil))
	require.NoError(t, err)

	require.NotContains(t, out.HTML, "СЛЕДУЮЩЕЕ ПО ПУТИ")
	require.Contains(t, out.HTML, "Первое слово")
}

func TestVerifyEmailRenders(t *testing.T) {
	out, err := newRenderer(t).Render(context.Background(), "verifyEmail", nil, map[string]any{
		"email":      "user@example.com",
		"code":       "481902",
		"ttlMinutes": 30,
		"verifyUrl":  "https://realtimemap.ru/verify?token=abc",
	})
	require.NoError(t, err)

	require.Contains(t, out.HTML, "481902")
	require.Contains(t, out.HTML, "user@example.com")
	require.Contains(t, out.HTML, "https://realtimemap.ru/verify?token=abc")
}

func TestPasswordResetRenders(t *testing.T) {
	out, err := newRenderer(t).Render(context.Background(), "passwordReset", nil, map[string]any{
		"username":    "winerty",
		"resetUrl":    "https://realtimemap.ru/reset?token=xyz",
		"ttlMinutes":  60,
		"requestedAt": "12 сент. 2026, 18:42",
		"device":      "iPhone · Safari",
		"expiresAt":   "19:42",
		"securityUrl": "https://realtimemap.ru/security",
	})
	require.NoError(t, err)

	require.Contains(t, out.HTML, "winerty")
	require.Contains(t, out.HTML, "https://realtimemap.ru/reset?token=xyz")
	require.Contains(t, out.HTML, "iPhone · Safari")
}

func TestNewSignInRenders(t *testing.T) {
	out, err := newRenderer(t).Render(context.Background(), "newSignIn", nil, map[string]any{
		"username":    "winerty",
		"signedInAt":  "12 сент. 2026, 18:40",
		"device":      "iPhone 15 · Safari",
		"location":    "Страсбург, Франция",
		"ipAddress":   "92.184.16.44",
		"sessionsUrl": "https://realtimemap.ru/security/sessions",
		"securityUrl": "https://realtimemap.ru/security",
	})
	require.NoError(t, err)

	require.Contains(t, out.HTML, "92.184.16.44")
	require.Contains(t, out.HTML, "Страсбург, Франция")
}

// Ни один шаблон не должен ссылаться на домен из макетов: адреса приходят из
// конфига, иначе смена домена потребует правки вёрстки.
func TestTemplatesHaveNoHardcodedDomain(t *testing.T) {
	p, err := NewProvider()
	require.NoError(t, err)

	for _, id := range p.IDs() {
		tmpl, err := p.Get(context.Background(), id, nil)
		require.NoError(t, err)
		require.NotContains(t, tmpl.HTML, "realtimemap.app", "шаблон %q содержит домен из макета", id)
	}
}

// Отсутствие обязательного поля обязано остановить отправку: без проверки
// html/template подставит пустую строку и письмо уйдёт с «Здравствуйте, !».
func TestRenderFailsOnMissingRequiredField(t *testing.T) {
	_, err := newRenderer(t).Render(context.Background(), "commentReply", nil, map[string]any{
		"username":   "winerty",
		"authorName": "Кто-то",
	})
	require.Error(t, err)
	require.True(t,
		strings.Contains(err.Error(), "commentText") || strings.Contains(err.Error(), "commentUrl"),
		"ошибка должна называть недостающее поле, получено: %v", err,
	)
}

// achievementData собирает полный набор полей достижения, позволяя тесту
// переопределить только интересующую его часть.
func achievementData(extra map[string]any) map[string]any {
	data := map[string]any{
		"username":                "winerty",
		"achievementTitle":        "Первое слово",
		"achievementDesc":         "вы оставили свой первый комментарий.",
		"achievementCondition":    "1 комментарий",
		"threshold":               1,
		"unlockedAt":              "12 сентября 2026, 18:40",
		"unlockedCount":           1,
		"totalCount":              4,
		"statPrimary":             "1",
		"statPrimaryLabel":        "КОММЕНТАРИЕВ",
		"statSecondary":           "10",
		"statSecondaryLabel":      "ОПЫТА",
		"achievementsUrl":         "https://realtimemap.ru/profile/achievements",
		"shareUrl":                "https://realtimemap.ru/profile/achievements/share",
		"notificationSettingsUrl": "https://realtimemap.ru/settings/notifications",
		"unsubscribeUrl":          "https://realtimemap.ru/unsubscribe",
	}
	maps.Copy(data, extra)
	return data
}
