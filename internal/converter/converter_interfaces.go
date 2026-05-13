package converter

import "github.com/gitwalk-m/conf2ext/internal/config"

type Converter interface {
	Convert(cfg *config.Configuration) error
}
