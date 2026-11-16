package presentation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallationIdentitySurvivesDisplayBrandChange(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	cases := map[string][]string{
		"scripts/build/package.py": {"Package: infradesktop", "Source: infradesktop"},
		"packaging/linux/infradesktop.desktop": {
			"Icon=infradesktop\n", "Exec=/usr/bin/infradesktop\n", "StartupWMClass=infradesktop\n",
		},
		"packaging/linux/infradesktopd@.service":                {"/usr/lib/infradesktop/infradesktopd"},
		"packaging/linux/org.infradesktop.reset-account.policy": {`id="org.infradesktop.reset-account"`, "/usr/lib/infradesktop/reset-account"},
		"wails.json": {`"outputfilename": "infradesktop"`},
	}
	for file, required := range cases {
		data, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range required {
			if !strings.Contains(string(data), value) {
				t.Errorf("%s lost installation identifier %q", file, value)
			}
		}
	}
}
