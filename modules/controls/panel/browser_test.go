package panel

import (
	"testing"

	"github.com/almerlucke/psalm/modules/controls/panel/paneltest"
)

// The page tests drive the panel's page in headless Chrome (paneltest)

func startBrowser(t *testing.T) *paneltest.Browser { return paneltest.Start(t) }

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	paneltest.Eventually(t, what, ok)
}

func TestMain(m *testing.M) { paneltest.Main(m) }
