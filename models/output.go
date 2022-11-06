package models

type Output struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	AsTempFile bool   `json:"as_temp_file,omitempty"` // By default as stream object
}
