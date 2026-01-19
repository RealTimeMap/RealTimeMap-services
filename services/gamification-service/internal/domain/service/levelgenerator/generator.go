package levelgenerator

type LevelGenerator interface {

	// CalculateExpForLevel вычисляет опыт для следуйщего уровня
	CalculateExpForLevel(level uint) uint

	// GetName название стратегии
	GetName() string
}
