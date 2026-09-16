package levelgenerator

// QuadraticGenerator наращивает стоимость каждого следующего уровня.
//
// Линейная стратегия давала постоянную дельту: XP в модели накопительный
// (см. model.Level), поэтому base*level означает ровно base опыта на любой
// переход. Десятый уровень стоил столько же, сколько второй, и прогресс не
// замедлялся никогда.
//
// Здесь дельта растёт на growth с каждым уровнем:
//
//	стоимость перехода на уровень L = base + growth*(L-2)
//	total(L) = base*(L-1) + growth*(L-1)*(L-2)/2
//
// При base=300 и growth=100 переход на 2-й стоит 300 XP, на 10-й — 1100.
type QuadraticGenerator struct {
	// baseExp — стоимость первого перехода (с 1-го уровня на 2-й).
	baseExp uint

	// growth — насколько дорожает каждый следующий переход.
	growth uint
}

func NewQuadraticGenerator() LevelGenerator {
	return &QuadraticGenerator{
		baseExp: 300,
		growth:  100,
	}
}

// CalculateExpForLevel возвращает накопительный опыт для уровня.
//
// Значение накопительное, а не за один переход: RecalculateLevel сравнивает
// его с progress.CurrentXP, который никогда не обнуляется.
func (q *QuadraticGenerator) CalculateExpForLevel(level uint) uint {
	// Первый уровень выдаётся стартовым: опыта он не стоит.
	if level <= 1 {
		return 0
	}

	// n — число переходов, пройденных к этому уровню.
	n := level - 1

	return q.baseExp*n + q.growth*n*(n-1)/2
}

func (q *QuadraticGenerator) GetName() string {
	return "QuadraticGenerator"
}
