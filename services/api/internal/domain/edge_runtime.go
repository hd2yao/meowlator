package domain

import "fmt"

type EdgeRuntime struct {
	Engine        string `json:"engine"`
	ModelVersion  string `json:"modelVersion"`
	LoadMS        int    `json:"loadMs"`
	InferMS       int    `json:"inferMs"`
	DeviceModel   string `json:"deviceModel"`
	FailureReason string `json:"failureReason,omitempty"`
}

func (e *EdgeRuntime) Validate() error {
	if e == nil {
		return nil
	}
	if e.Engine == "" {
		return fmt.Errorf("edgeRuntime.engine is required")
	}
	if e.ModelVersion == "" {
		return fmt.Errorf("edgeRuntime.modelVersion is required")
	}
	if e.LoadMS < 0 || e.InferMS < 0 {
		return fmt.Errorf("edgeRuntime timings must be >= 0")
	}
	if e.DeviceModel == "" {
		return fmt.Errorf("edgeRuntime.deviceModel is required")
	}
	return nil
}

type EdgeMeta struct {
	Engine         string `json:"engine"`
	ModelVersion   string `json:"modelVersion"`
	LoadMS         int    `json:"loadMs"`
	InferMS        int    `json:"inferMs"`
	DeviceModel    string `json:"deviceModel"`
	FailureReason  string `json:"failureReason,omitempty"`
	FallbackUsed   bool   `json:"fallbackUsed"`
	UsedEdgeResult bool   `json:"usedEdgeResult"`
}
