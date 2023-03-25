package message

type Metrics struct {
	ID    string   `json:"id"`              // Имя метрики
	MType string   `json:"type"`            // Параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // Значение метрики в случае передачи gauge
}

// todo: по хорошему добавить проверку от дурака на вход.значения value и тип MType
// todo: надо! иначе паника
//
//	либо удалить вообще
func NewMetrics(ID string, MType string, value interface{}) *Metrics {
	metric := Metrics{ID: ID, MType: MType}
	if MType == "gauge" {
		floatVal := value.(float64)
		metric.Value = &floatVal
	} else if MType == "counter" {
		intVal := value.(int64)
		metric.Delta = &intVal
	}
	return &metric
}
