package models

type Param struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	IsInput    bool   `json:"is_input,omitempty"`
	AsTempFile bool   `json:"as_temp_file,omitempty"`
}
