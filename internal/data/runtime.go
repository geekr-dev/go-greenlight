package data

import (
	"encoding/json"
	"fmt"
)

func (m Movie) MarshalJSON() ([]byte, error) {
	var runtime string

	if m.Runtime != 0 {
		runtime = fmt.Sprintf("%d mins", m.Runtime)
	}

	type MovieAlias Movie

	aux := struct {
		MovieAlias
		Runtime string `json:"runtime"`
	}{
		MovieAlias: MovieAlias(m),
		Runtime:    runtime,
	}

	return json.Marshal(aux)
}
