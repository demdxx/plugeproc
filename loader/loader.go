package loader

import "github.com/demdxx/plugeproc/models"

type Loader interface {
	Load() ([]*models.Info, error)
}
