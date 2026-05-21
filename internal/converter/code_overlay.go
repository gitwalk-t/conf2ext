package converter

import (
	"fmt"
	"log"

	"github.com/gitwalk-m/conf2ext/internal/codeoverlay"
	"github.com/gitwalk-m/conf2ext/internal/config"
)

func applyCodeOverlayIfEnabled(cfg *config.Configuration, rootPath string) error {
	if cfg == nil || !cfg.IsCodeOverlayEnabled() {
		return nil
	}

	overlayPath, err := codeoverlay.ResolveRuntimeOverlayPath(cfg)
	if err != nil {
		return fmt.Errorf("не удалось определить путь к code overlay artifact: %w", err)
	}

	diagnostics, err := codeoverlay.Apply(rootPath, overlayPath)
	if err != nil {
		return fmt.Errorf("не удалось применить code overlay %s: %w", overlayPath, err)
	}

	log.Printf(
		"code overlay: loaded=%d applied=%d skipped=%d missing=%d fallback=%d conflicted=%d file=%s",
		diagnostics.Loaded,
		diagnostics.Applied,
		diagnostics.Skipped,
		diagnostics.Missing,
		diagnostics.Fallback,
		diagnostics.Conflicted,
		overlayPath,
	)

	return nil
}
