package domain

type Record struct {
	ID             string
	CollectionName string
	Vector         []float32
	Data           map[string]float32
}

func NormalizeNumber(number float32, rng MinMaxRange) float32 {
	if rng.Min == rng.Max {
		return 0
	}
	result := (number - rng.Min) / (rng.Max - rng.Min)
	if result > 1 {
		return 1
	}
	if result < 0 {
		return 0
	}
	return result
}

func NormalizeBool(value bool) float32 {
	if value {
		return 1
	}

	return 0
}

func NewRecord(id, collectionName string, vector []float32, data map[string]float32) *Record {
	return &Record{
		ID:             id,
		CollectionName: collectionName,
		Vector:         vector,
		Data:           data,
	}
}
