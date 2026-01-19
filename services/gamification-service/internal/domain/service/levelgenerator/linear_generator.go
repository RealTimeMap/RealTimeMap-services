package levelgenerator

type LinearGenerator struct {
	baseExp uint
}

func NewLinearGenerator() LevelGenerator {
	return &LinearGenerator{
		baseExp: 150,
	}
}

func (l *LinearGenerator) CalculateExpForLevel(level uint) uint {
	return l.baseExp * level
}

func (l *LinearGenerator) GetName() string {
	return "LinearGenerator"
}
